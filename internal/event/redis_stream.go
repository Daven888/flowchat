package event

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Daven888/flowchat/pkg/logger"
	appredis "github.com/Daven888/flowchat/pkg/redis"
)

const (
	StreamKey       = "flowchat:model_call_events"
	StreamDLQKey    = "flowchat:model_call_events:dlq"
	ConsumerGroup   = "flowchat-workers"
	PendingIdleTime = 60 * time.Second
	MaxRetry        = 3
	BlockTimeout    = 5 * time.Second
)

// consumerName returns a unique consumer name: hostname + pid.
func consumerName() string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

// Publisher publishes events to the Redis Stream.
type Publisher struct {
	client *redis.Client
}

// NewPublisher creates a new Publisher.
func NewPublisher() *Publisher {
	return &Publisher{client: appredis.Client}
}

// Publish sends a ModelCallFinishedEvent to the stream. Returns an error if Redis is
// unreachable; the caller should log and continue — publishing must never fail the main flow.
func (p *Publisher) Publish(ctx context.Context, event *ModelCallFinishedEvent) error {
	data, err := event.Marshal()
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{
			"event": string(data),
		},
	}).Err()
}

// PublishDLQ sends a dead letter event to the DLQ stream.
func (p *Publisher) PublishDLQ(ctx context.Context, event *DeadLetterEvent) error {
	data, err := event.Marshal()
	if err != nil {
		return fmt.Errorf("marshal dlq event: %w", err)
	}
	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamDLQKey,
		Values: map[string]interface{}{
			"event": string(data),
		},
	}).Err()
}

// Consumer reads events from the Redis Stream using consumer groups, processes them,
// handles retries, and routes exhausted events to the DLQ.
type Consumer struct {
	client  *redis.Client
	name    string
	handler EventHandler
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

// EventHandler processes a ModelCallFinishedEvent. Return nil on success (ACK),
// non-nil on failure (no ACK, message stays pending for retry).
type EventHandler func(ctx context.Context, event *ModelCallFinishedEvent) error

// NewConsumer creates a new Consumer.
func NewConsumer(handler EventHandler) *Consumer {
	return &Consumer{
		client:  appredis.Client,
		name:    consumerName(),
		handler: handler,
	}
}

// Start initializes the consumer group and begins background processing.
// The consumer group is created with MKSTREAM so the stream is created if it
// doesn't exist. If the group already exists, the error is ignored (not fatal).
func (c *Consumer) Start(ctx context.Context) error {
	// Create consumer group (idempotent — ignore BUSYGROUP error).
	err := c.client.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "0").Err()
	if err != nil && !redis.HasErrorPrefix(err, "BUSYGROUP") {
		return fmt.Errorf("create consumer group: %w", err)
	}

	ctx, c.cancel = context.WithCancel(ctx)

	// Main consumption loop: read new messages.
	c.wg.Add(1)
	go c.consumeNew(ctx)

	// Pending claim loop: periodically claim and retry idle pending messages.
	c.wg.Add(1)
	go c.claimPending(ctx)

	if logger.Log != nil {
		logger.Log.Info("Redis Stream consumer started",
			zap.String("stream", StreamKey),
			zap.String("group", ConsumerGroup),
			zap.String("consumer", c.name),
			zap.Int("max_retry", MaxRetry),
			zap.Duration("pending_idle", PendingIdleTime),
		)
	}
	return nil
}

// Stop gracefully shuts down the consumer.
func (c *Consumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	if logger.Log != nil {
		logger.Log.Info("Redis Stream consumer stopped")
	}
}

// consumeNew reads new messages from the stream using XREADGROUP.
func (c *Consumer) consumeNew(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		entries, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    ConsumerGroup,
			Consumer: c.name,
			Streams:  []string{StreamKey, ">"},
			Count:    10,
			Block:    BlockTimeout,
		}).Result()

		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			if logger.Log != nil {
				logger.Log.Warn("XREADGROUP error, retrying", zap.Error(err))
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range entries {
			for _, msg := range stream.Messages {
				c.processMessage(ctx, msg)
			}
		}
	}
}

// claimPending periodically claims idle pending messages from other consumers
// and retries them. Uses XAUTOCLAIM for efficient claiming.
func (c *Consumer) claimPending(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.autoClaim(ctx)
		}
	}
}

// autoClaim claims idle pending messages and retries them.
func (c *Consumer) autoClaim(ctx context.Context) {
	for {
		msgs, nextStart, err := c.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   StreamKey,
			Group:    ConsumerGroup,
			Consumer: c.name,
			MinIdle:  PendingIdleTime,
			Start:    "0",
			Count:    10,
		}).Result()

		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, redis.Nil) {
				return
			}
			if logger.Log != nil {
				logger.Log.Warn("XAUTOCLAIM error", zap.Error(err))
			}
			return
		}

		if len(msgs) == 0 {
			return
		}

		for _, msg := range msgs {
			c.processMessage(ctx, msg)
		}

		// Continue if there might be more pending messages.
		if nextStart == "0" || nextStart == "" {
			return
		}
	}
}

// processMessage handles a single stream message: parse, execute handler, ACK or retry.
func (c *Consumer) processMessage(ctx context.Context, msg redis.XMessage) {
	eventData, ok := msg.Values["event"]
	if !ok {
		// Malformed message — ACK to skip.
		c.ack(ctx, msg.ID)
		return
	}

	data, ok := eventData.(string)
	if !ok {
		c.ack(ctx, msg.ID)
		return
	}

	event, err := UnmarshalModelCallEvent([]byte(data))
	if err != nil {
		if logger.Log != nil {
			logger.Log.Warn("failed to unmarshal event, skipping",
				zap.String("message_id", msg.ID),
				zap.Error(err),
			)
		}
		c.ack(ctx, msg.ID)
		return
	}

	if logger.Log != nil {
		logger.Log.Debug("processing event",
			zap.String("request_id", event.RequestID),
			zap.String("message_id", msg.ID),
			zap.Int("retry_count", event.RetryCount),
		)
	}

	// Process the event.
	if err := c.handler(ctx, event); err != nil {
		event.RetryCount++

		if event.RetryCount > MaxRetry {
			// Exhausted retries — move to DLQ.
			c.moveToDLQ(ctx, msg.ID, event, err.Error())
			return
		}

		// ACK original + re-publish with incremented retry_count.
		c.ack(ctx, msg.ID)
		if pubErr := NewPublisher().Publish(ctx, event); pubErr != nil {
			if logger.Log != nil {
				logger.Log.Error("failed to re-publish event for retry",
					zap.String("request_id", event.RequestID),
					zap.Int("retry_count", event.RetryCount),
					zap.Error(pubErr),
				)
			}
		} else {
			if logger.Log != nil {
				logger.Log.Warn("event processing failed, scheduled retry",
					zap.String("request_id", event.RequestID),
					zap.Int("retry_count", event.RetryCount),
					zap.Error(err),
				)
			}
		}
		return
	}

	// Success — ACK the message.
	c.ack(ctx, msg.ID)

	if logger.Log != nil {
		logger.Log.Debug("event processed successfully",
			zap.String("request_id", event.RequestID),
			zap.String("message_id", msg.ID),
		)
	}
}

// moveToDLQ moves an exhausted event to the dead letter stream and ACKs the original.
func (c *Consumer) moveToDLQ(ctx context.Context, msgID string, event *ModelCallFinishedEvent, reason string) {
	dlq := &DeadLetterEvent{
		OriginalEvent:   *event,
		FailedReason:    reason,
		DeadLetterAt:    time.Now().UnixMilli(),
		FinalRetryCount: event.RetryCount,
	}

	if err := NewPublisher().PublishDLQ(ctx, dlq); err != nil {
		if logger.Log != nil {
			logger.Log.Error("failed to publish DLQ event",
				zap.String("request_id", event.RequestID),
				zap.String("message_id", msgID),
				zap.Error(err),
			)
		}
		// Still ACK the original — we won't retry forever.
	} else {
		if logger.Log != nil {
			logger.Log.Warn("event moved to DLQ",
				zap.String("request_id", event.RequestID),
				zap.String("failed_reason", reason),
				zap.Int("final_retry_count", event.RetryCount),
			)
		}
	}

	c.ack(ctx, msgID)
}

// ack acknowledges a stream message.
func (c *Consumer) ack(ctx context.Context, msgID string) {
	if err := c.client.XAck(ctx, StreamKey, ConsumerGroup, msgID).Err(); err != nil {
		if logger.Log != nil {
			logger.Log.Warn("failed to ACK message",
				zap.String("message_id", msgID),
				zap.Error(err),
			)
		}
	}
}
