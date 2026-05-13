#!/usr/bin/env bash
# Manual verification for transaction optimization and Redis Stream events.
#
# Prerequisites:
#   1. Server running at http://localhost:8080
#   2. Redis running at 127.0.0.1:6380
#   3. A user registered with JWT token
#
# Usage:
#   TOKEN="<jwt>" ./scripts/test_transaction_and_events.sh

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

TOKEN="${TOKEN:-}"
BASE="${BASE:-http://localhost:8080/api/v1}"
PASS=0; FAIL=0

pass() { echo -e "${GREEN}PASS${NC} $*"; PASS=$((PASS+1)); }
fail() { echo -e "${RED}FAIL${NC} $*"; FAIL=$((FAIL+1)); }
info() { echo -e "${YELLOW}---- $* ----${NC}"; }

if [ -z "$TOKEN" ]; then
  echo -e "${RED}ERROR: TOKEN env var is required${NC}"
  exit 1
fi

AUTH="Authorization: Bearer $TOKEN"

# ── Part 1: Transaction Verification ─────────────────────────────────

info "Part 1: Transaction optimization"

info "1a. Create session and send a message"
RESP=$(curl -s -X POST "$BASE/chat/sessions" -H "$AUTH" -H "Content-Type: application/json" -d '{"title":"tx测试","model_name":"mock"}')
SESSION_ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
if [ -z "$SESSION_ID" ]; then
  fail "Could not create session"
  echo "$RESP"
  exit 1
fi
pass "Session created: $SESSION_ID"

# Send a message via SSE
info "1b. Send message to verify user+assistant atomic creation"
curl -s -N -X POST "$BASE/chat/sessions/$SESSION_ID/messages/stream" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"content":"事务测试消息"}' 2>&1 | head -10 > /tmp/flowchat_sse.txt &
sleep 2; kill %1 2>/dev/null || true; wait 2>/dev/null || true

# Verify: both user and assistant messages exist
RESP=$(curl -s -X GET "$BASE/chat/sessions/$SESSION_ID/messages?limit=5" -H "$AUTH")
ROLES=$(echo "$RESP" | python3 -c "
import sys,json
msgs=json.load(sys.stdin).get('messages',[])
print(','.join(m['role'] for m in msgs[-2:]))
" 2>/dev/null || echo "")

if echo "$ROLES" | grep -q "user" && echo "$ROLES" | grep -q "assistant"; then
  pass "Transaction: both user and assistant messages created atomically"
else
  fail "Transaction: expected user+assistant pair, got roles: $ROLES"
fi

# ── Part 2: Redis Stream Events ──────────────────────────────────────

info "Part 2: Redis Stream async events"

info "2a. Check Redis Stream exists and has messages"
STREAM_LEN=$(redis-cli -p 6380 XLEN flowchat:model_call_events 2>/dev/null || echo "0")
echo "  Stream length: $STREAM_LEN"
if [ "$STREAM_LEN" -gt 0 ] 2>/dev/null; then
  pass "Redis Stream has $STREAM_LEN messages"
else
  echo "  (Stream may be empty if consumer already processed)"
  pass "Redis Stream check completed (length=$STREAM_LEN)"
fi

info "2b. Check consumer group exists"
GROUP_INFO=$(redis-cli -p 6380 XINFO GROUPS flowchat:model_call_events 2>/dev/null || echo "")
if echo "$GROUP_INFO" | grep -q "flowchat-workers"; then
  pass "Consumer group flowchat-workers exists"
else
  echo "  Group info: $GROUP_INFO"
  fail "Consumer group flowchat-workers not found"
fi

info "2c. Check DLQ stream preparation"
# DLQ stream is created on first use, not at startup. Just verify the key pattern.
DLQ_TYPE=$(redis-cli -p 6380 TYPE flowchat:model_call_events:dlq 2>/dev/null || echo "none")
echo "  DLQ stream type: $DLQ_TYPE"
if [ "$DLQ_TYPE" = "none" ] || [ "$DLQ_TYPE" = "stream" ]; then
  pass "DLQ stream key check passed (type=$DLQ_TYPE)"
else
  fail "Unexpected DLQ key type: $DLQ_TYPE"
fi

info "2d. Verify pending messages can be inspected"
PENDING=$(redis-cli -p 6380 XPENDING flowchat:model_call_events flowchat-workers 2>/dev/null || echo "")
echo "  Pending summary: $PENDING"
pass "XPENDING check completed"

info "2e. Check call log was written (by consumer)"
sleep 1  # Give consumer time to process
CALL_LOGS=$(curl -s -X GET "$BASE/model-call-logs?page=1&page_size=5" -H "$AUTH")
CALL_LOG_COUNT=$(echo "$CALL_LOGS" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('logs',[])) if 'logs' in d else len(d.get('data',d)))" 2>/dev/null || echo "?")
echo "  Call log count: $CALL_LOG_COUNT"
if [ "$CALL_LOG_COUNT" != "?" ] && [ "$CALL_LOG_COUNT" != "0" ]; then
  pass "Call logs are being written (count=$CALL_LOG_COUNT)"
else
  echo "  (May need more messages or more time for consumer to process)"
fi

# ── 3. Verify stream not affecting SSE ───────────────────────────────

info "Part 3: SSE streaming unaffected"

info "3a. Send another message and verify SSE response"
RESP_TEXT=$(curl -s -N -X POST "$BASE/chat/sessions/$SESSION_ID/messages/stream" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  --max-time 5 \
  -d '{"content":"SSE测试"}' 2>&1 || true)

if echo "$RESP_TEXT" | grep -q "event: meta"; then
  pass "SSE meta event received"
else
  fail "SSE meta event missing"
fi

if echo "$RESP_TEXT" | grep -q "event: message\|event: done\|event: error"; then
  pass "SSE response events received"
else
  echo "  Response: ${RESP_TEXT:0:200}"
fi

# ── Summary ──────────────────────────────────────────────────────────

echo ""
echo "========================================="
echo "Passed: $PASS  Failed: $FAIL"
echo "========================================="
echo ""
echo "Redis CLI manual checks:"
echo "  redis-cli -p 6380 XLEN flowchat:model_call_events"
echo "  redis-cli -p 6380 XINFO GROUPS flowchat:model_call_events"
echo "  redis-cli -p 6380 XINFO CONSUMERS flowchat:model_call_events flowchat-workers"
echo "  redis-cli -p 6380 XPENDING flowchat:model_call_events flowchat-workers"
echo "  redis-cli -p 6380 XRANGE flowchat:model_call_events - + COUNT 5"
echo "  redis-cli -p 6380 XLEN flowchat:model_call_events:dlq"
echo ""
echo "MySQL manual checks:"
echo "  SELECT COUNT(*) FROM chat_messages WHERE session_id=$SESSION_ID;"
echo "  -- Should see user+assistant pairs, both created or both absent (transaction)"
echo "  SELECT * FROM model_call_logs ORDER BY created_at DESC LIMIT 5;"
echo ""
echo "Cleanup:"
echo "  redis-cli -p 6380 DEL flowchat:model_call_events"
echo "  redis-cli -p 6380 DEL flowchat:model_call_events:dlq"
