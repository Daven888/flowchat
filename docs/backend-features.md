# FlowChat 后端功能详解

本文档描述各功能模块的设计、实现细节和数据模型。

---

## 1. 用户认证

### 实现

- 注册：用户名 + 邮箱 + 密码，密码使用 bcrypt 哈希存储
- 登录：邮箱 + 密码验证，签发 JWT Token
- 鉴权：JWT 中间件解析 Token，将 `user_id` 注入 Gin Context

### 相关文件

| 文件 | 职责 |
|------|------|
| `internal/model/user.go` | User 模型 |
| `internal/repository/user_repo.go` | 用户数据库操作 |
| `internal/service/user_service.go` | 注册/登录逻辑 |
| `internal/handler/auth_handler.go` | HTTP 处理器 |
| `internal/auth/jwt.go` | JWT Token 签发与解析 |
| `internal/middleware/auth.go` | JWT 鉴权中间件 |

### 数据库

```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL UNIQUE,
    email VARCHAR(128) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    status TINYINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

---

## 2. 用户级 Provider API Key 管理

### 设计动机

用户使用自己的 API Key 调用模型服务，而非使用服务端统一 Key。Key 在数据库中 AES-256-GCM 加密存储，前端只看到安全摘要信息。

### 加密方案

- 密钥派生：`FLOWCHAT_CREDENTIAL_SECRET` 环境变量 → SHA-256 → 32 字节 AES 密钥
- 加密算法：AES-256-GCM，每次加密随机生成 12 字节 nonce
- 存储格式：`base64(nonce) + ":" + base64(ciphertext)`，同一明文每次加密结果不同

### 凭据操作

- **Upsert**：插入或更新（status 重置为 1），API Key trim 后加密存储，记录 `key_suffix`（末 4 字符）
- **GetDecrypted**：按 user_id + provider_name 查询 active 凭据，解密后返回明文 Key
- **ListStatus**：遍历所有配置的 Provider，返回每种凭据的安全状态信息
- **Delete**：软删除（status 设为 0）

### mock provider 特殊处理

mock provider 不需要 API Key。对 mock 执行 Upsert 或 GetDecrypted 会返回特定错误，不会尝试加密/解密。

### 安全约束

- `GET /api/v1/user/credentials` 不返回 encrypted_api_key 或任何明文 Key
- `PUT` 响应只返回 `key_suffix`（末 4 字符），不返回完整 Key
- 解密后的明文 Key 只在内存中短暂存在（单次 `BeginStream` 调用），通过 `ChatRequest.APIKey` 传给 Provider
- Provider 实例不持有 API Key 引用
- Provider 配置文件中无 `api_key_env` 字段

### 相关文件

| 文件 | 职责 |
|------|------|
| `internal/model/user_provider_credential.go` | 凭据模型 |
| `internal/repository/user_provider_credential_repo.go` | 凭据数据库操作 |
| `internal/service/credential_service.go` | 加密/解密/校验逻辑 |
| `internal/handler/credential_handler.go` | HTTP 处理器 |
| `pkg/cryptoutil/aes.go` | AES-256-GCM 加解密实现 |

---

## 3. 多 Provider / 多模型配置

### Provider 配置

Provider 只包含非敏感连接信息：

```yaml
ai:
  providers:
    deepseek:
      type: openai_compatible
      base_url: https://api.deepseek.com
      display_name: DeepSeek
```

### 模型配置

每个模型可单独配置参数：

```yaml
models:
  - name: deepseek-chat
    provider: deepseek
    api_model: deepseek-chat
    enabled: true
    max_context_messages: 10
    timeout_seconds: 120
    max_retries: 1
    context_compression:
      enabled: true
      max_messages_before_compress: 30
```

`name` 是用户创建会话时传入的模型名。`api_model` 是请求上游 API 时使用的名称。

### Provider 注册

`ProviderRegistry` 维护 provider name → ChatProvider 实例的映射。mock provider 内置注册，其他 provider 从配置文件构建。

### 相关文件

| 文件 | 职责 |
|------|------|
| `internal/config/config.go` | 配置结构定义 |
| `internal/service/model_service.go` | 模型查询 + Provider 注册 |
| `internal/provider/provider.go` | ChatProvider 接口定义 |

---

## 4. 会话管理

### 生命周期

- **创建**：校验 model_name 存在且 enabled，若 title 为空则使用默认值 "新的对话"
- **列表**：仅返回 status=1（active）的会话
- **详情**：校验归属当前用户且 status=1
- **软删除**：status 设为 0

### 数据模型

```sql
CREATE TABLE chat_sessions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    title VARCHAR(128) NOT NULL,
    model_name VARCHAR(64) NOT NULL,
    status TINYINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_user_updated (user_id, updated_at)
);
```

### 相关文件

| 文件 | 职责 |
|------|------|
| `internal/model/session.go` | ChatSession 模型 |
| `internal/repository/session_repo.go` | 会话数据库操作 |
| `internal/service/session_service.go` | 会话业务逻辑 |
| `internal/handler/session_handler.go` | HTTP 处理器 |

---

## 5. 消息生命周期

### 状态流转

```
User Message:  (创建时) completed
Assistant Message:
  placeholder → generating
              → completed  (成功生成)
              → failed     (生成出错)
              → cancelled  (连接断开)
```

### 事务写入

`BeginStream` 中保存 user 消息（completed）和创建 assistant 占位消息（generating）在同一个 GORM 事务中执行。两个 INSERT 要么都成功，要么都失败回滚，避免只保存了 user 消息但 assistant 占位创建失败的半成品状态。

事务方法：`MessageRepository.CreateUserAndAssistantMessagesTx`

### 角色说明

- `user`：用户消息
- `assistant`：模型回复
- `system`：仅用于上下文压缩中的摘要消息（向 Provider 传递），不存入 chat_messages

### 数据模型

```sql
CREATE TABLE chat_messages (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    session_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role VARCHAR(32) NOT NULL,
    content MEDIUMTEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'completed',
    error_message VARCHAR(255) NULL,
    token_count INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    INDEX idx_session_id_id (session_id, id),
    INDEX idx_user_created (user_id, created_at)
);
```

### 相关文件

| 文件 | 职责 |
|------|------|
| `internal/model/message.go` | ChatMessage 模型 + 角色/状态常量 |
| `internal/repository/message_repo.go` | 消息数据库操作（含事务方法） |
| `internal/service/message_service.go` | 消息 CRUD + 翻页 + 搜索 |
| `internal/handler/message_handler.go` | HTTP 处理器 |

---

## 6. SSE 流式对话主链路

### 流程

```
1. JWT 鉴权
2. 解析参数，校验 model_name 存在且 enabled
3. 敏感词过滤（大小写不敏感子串匹配）
4. 获取用户 API Key（mock 跳过）
5. 生成 request_id，获取 Redis 会话锁（SETNX）
6. 事务写入 user + assistant 消息
7. 上下文组装（压缩或最近 N 条）
8. 调用 Provider.StreamChat（HTTP SSE 上游或 Mock 本地生成）
9. SSE 返回：meta → message... → done/error
10. 内存聚合完整回复，更新 assistant 消息为 completed
11. 释放 Redis 锁
12. defer 发布 ModelCallFinishedEvent 到 Redis Stream
```

### Provider 超时与重试

- 使用 `context.WithTimeout` 控制调用超时
- 支持配置 `max_retries`，失败时线性退避重试（200ms × attempt）
- 重试仅针对初始连接失败，流式传输中途失败不重试

### 敏感词过滤

- 在 `BeginStream` 之前检查
- 命中时返回 `error_code=SENSITIVE_CONTENT`，不保存消息、不调用 Provider
- 简单子串匹配，大小写不敏感

### 会话生成锁

- Key：`flowchat:session_lock:{session_id}`
- 使用 Redis SETNX 加锁，Lua 脚本原子释放（检查 request_id 匹配）
- TTL 可配置（默认 180s），防止死锁
- 锁持有者在 `BeginStream` 成功后转移给调用方，由 `ChatHandler.Stream` 的 defer 释放

### 相关文件

| 文件 | 职责 |
|------|------|
| `internal/handler/chat_handler.go` | SSE 处理器，编排整个流式流程 |
| `internal/service/chat_service.go` | ChatService：会话校验、锁、消息保存、Provider 调用、完成/失败/取消 |
| `internal/lock/manager.go` | Redis 分布式锁 |
| `internal/sensitive/filter.go` | 敏感词过滤 |

---

## 7. Provider 抽象

### 接口定义

```go
type ChatProvider interface {
    StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}

type ChatRequest struct {
    RequestID string
    ModelName string
    Messages  []ProviderMessage
    APIKey    string
}

type ProviderMessage struct {
    Role    string
    Content string
}

type ChatChunk struct {
    Content          string
    Done             bool
    Err              error
    FinishReason     string
    PromptTokens     int
    CompletionTokens int
}
```

### Mock Provider

- 本地生成模拟回复：`"这是 Mock Provider 生成的回复。你刚才的问题是：{userMsg}"`
- 每 100ms 输出 5 个 rune，模拟流式效果
- 检测首条消息为 system role 时进入摘要模式，返回固定摘要文本
- 不需要 API Key

### OpenAI Compatible Provider

- 发送 POST 请求到 `{base_url}/chat/completions`
- 读取 SSE 流（`data: {...}` 行），解析 `delta.content`
- 监听 `[DONE]` 标记
- `ChatRequest.APIKey` 作为 `Authorization: Bearer` Header
- Token 估算使用 rune count

### 相关文件

| 文件 | 职责 |
|------|------|
| `internal/provider/provider.go` | 接口 + 通用类型定义 |
| `internal/provider/mock/provider.go` | Mock Provider 实现 |
| `internal/provider/openai/provider.go` | OpenAI Compatible Provider 实现 |

---

## 8. 长会话能力

### 消息分页

使用游标分页（cursor-based pagination）：

- **参数**：`before_id`（上一页最旧消息的 ID）+ `limit`（最大 100）
- **查询**：`WHERE session_id = ? AND id < ? ORDER BY id DESC LIMIT ?+1`
- **has_more**：取 limit+1 条，若实际返回超过 limit 条则为 true
- **next_before_id**：返回页中第一条（最旧）消息的 ID
- **ASC 排序**：DB 查询 DESC 后，在 service 层 `reverseMessages` 翻转为 ASC

### 会话内搜索

- 使用 MySQL `LIKE '%keyword%'` 匹配 content
- 限定 `WHERE session_id = ?`，不跨会话搜索
- 结果按 id ASC 排序

### 上下文压缩

当 session 的 completed 消息数超过 `max_messages_before_compress` 阈值时触发：

1. 取所有 completed 消息，排除最近 `max_context_messages` 条
2. 将较早消息格式化为 `用户：...\n助手：...\n` 的文本
3. 调用当前 Provider 生成摘要（system prompt 要求总结需求/事实/约束/已完成/待解决）
4. 摘要持久化到 `chat_session_summaries`，记录 `last_message_id`
5. 后续请求上下文为：`[system summary] + [最近 N 条 completed 消息]`

**特性：**
- 摘要存储在独立表，不出现在消息列表、搜索、Markdown 导出中
- 只压缩 status=completed 的消息，generating/failed/cancelled 不参与
- 生成失败时降级为纯最近 N 条消息，不影响用户聊天

**数据模型：**
```sql
CREATE TABLE chat_session_summaries (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    session_id BIGINT NOT NULL,
    content MEDIUMTEXT NOT NULL,
    last_message_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE KEY uk_session_id (session_id)
);
```

### 相关文件

| 文件 | 职责 |
|------|------|
| `internal/repository/message_repo.go` | 分页查询 / 搜索 / 计数 |
| `internal/service/message_service.go` | 分页逻辑 / 搜索 / 上下文读取 |
| `internal/model/summary.go` | ChatSummary 模型 |
| `internal/repository/summary_repo.go` | 摘要 Upsert / 查询 |
| `internal/service/compression_service.go` | 压缩触发判断 / 摘要生成 / 上下文组装 |

---

## 9. 异步旁路任务

### 设计动机

将非核心链路的旁路任务（写调用日志、更新统计、自动标题）从 SSE 响应的同步 defer 中移出，改为通过 Redis Stream 异步处理。这样：
- SSE 响应延迟不受旁路任务影响
- 旁路任务失败可重试，不影响主聊天流程
- 支持多实例消费（consumer group 负载均衡）

### 事件定义

`ModelCallFinishedEvent` 包含一次模型调用完成后的所有信息：

- request_id, user_id, session_id, provider, model_name, status
- prompt_tokens, completion_tokens, latency_ms
- error_code, error_message, finish_reason
- started_at, finished_at, title_source_content
- retry_count

注意：事件 **不包含** API Key、加密凭据、完整 assistant 回复。

### Redis Stream 设计

```
Stream Key:       flowchat:model_call_events
Consumer Group:   flowchat-workers
DLQ Stream Key:   flowchat:model_call_events:dlq
Max Retry:        3
Pending Idle:     60s
Block Timeout:    5s
Consumer Name:    {hostname}-{pid}
```

### 生产者

`ChatHandler.Stream` 的 defer 中发布事件：

```go
defer func() {
    ev := &event.ModelCallFinishedEvent{...}
    h.eventPublisher.Publish(ctx, ev)  // 发布失败只记日志，不影响 SSE
}()
```

### 消费者

服务启动时创建 consumer group（`XGROUP CREATE ... MKSTREAM`）并启动两个 goroutine：

1. **consumeNew**：`XREADGROUP` 读取新消息（`>`），处理成功后 `XACK`
2. **claimPending**：每 10s 执行 `XAUTOCLAIM` 认领 idle >60s 的 pending 消息

### 重试与 DLQ

处理失败时：
1. `event.RetryCount++`
2. `XACK` 原消息
3. `XADD` 新消息（带 incremented retry_count）
4. 若 `RetryCount > 3`：构造 `DeadLetterEvent`（含原始事件 + 失败原因 + 时间戳），`XADD` 到 DLQ Stream，`XACK` 原消息

### 消费者处理逻辑

```text
收到 ModelCallFinishedEvent
  → 写入 model_call_logs
  → 更新 user_model_usage_stats
  → 若 status=success 且 title 仍为默认值 → 更新会话标题
```

任一步失败返回 error，触发重试。标题生成失败不返回 error（非关键任务）。

### 相关文件

| 文件 | 职责 |
|------|------|
| `internal/event/model_call_event.go` | 事件结构 + JSON 序列化 |
| `internal/event/redis_stream.go` | Publisher / Consumer / DLQ |
| `internal/event/model_call_handler.go` | 事件处理（写日志/统计/标题） |

---

## 10. 数据库核心表总览

| 表名 | 用途 | 关键索引 |
|------|------|----------|
| `users` | 用户 | username UNIQUE, email UNIQUE |
| `chat_sessions` | 会话 | user_id, (user_id, updated_at) |
| `chat_messages` | 消息 | (session_id, id), (user_id, created_at) |
| `user_provider_credentials` | API Key 凭据 | (user_id, provider_name) UNIQUE, user_id |
| `chat_session_summaries` | 上下文摘要 | session_id UNIQUE |
| `model_call_logs` | 调用日志 | request_id UNIQUE, (user_id, created_at), session_id |
| `user_model_usage_stats` | 用量统计 | (user_id, model_name, stat_date) UNIQUE, (user_id, stat_date) |

完整 DDL 见 `scripts/init.sql`。

---

## 11. 配置项清单

| 配置路径 | 类型 | 默认值 | 说明 |
|----------|------|--------|------|
| `server.port` | int | 8080 | HTTP 端口 |
| `mysql.*` | - | - | MySQL 连接参数 |
| `redis.*` | - | - | Redis 连接参数 |
| `jwt.secret` | string | - | JWT 签名密钥 |
| `jwt.expire_hours` | int | 24 | Token 过期时间 |
| `chat.session_lock_ttl_seconds` | int | 180 | 会话锁 TTL |
| `chat.max_message_length` | int | 4000 | 消息最大长度 |
| `credential.encryption_key_env` | string | - | 加密密钥环境变量名 |
| `ai.providers.{name}.type` | string | - | Provider 类型 |
| `ai.providers.{name}.base_url` | string | - | 模型服务地址 |
| `ai.providers.{name}.display_name` | string | - | 前端展示名称 |
| `models[].name` | string | - | 对外模型名 |
| `models[].provider` | string | - | 关联 Provider |
| `models[].api_model` | string | - | 上游 API 模型名 |
| `models[].enabled` | bool | false | 是否启用 |
| `models[].max_context_messages` | int | 10 | 上下文消息数 |
| `models[].timeout_seconds` | int | 120 | 调用超时 |
| `models[].max_retries` | int | 0 | 重试次数 |
| `models[].context_compression.enabled` | bool | false | 是否启用压缩 |
| `models[].context_compression.max_messages_before_compress` | int | 0 | 压缩触发阈值 |
| `sensitive_words` | []string | [] | 敏感词列表 |
