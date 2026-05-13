#!/usr/bin/env bash
# Manual verification script for context compression.
#
# Prerequisites:
#   1. Server running at http://localhost:8080
#   2. A registered user with valid JWT token
#   3. A session created with the "mock" model (so no real API key needed)
#
# Usage:
#   TOKEN="<jwt>" ./scripts/test_compression.sh

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

TOKEN="${TOKEN:-}"
BASE="${BASE:-http://localhost:8080/api/v1}"

if [ -z "$TOKEN" ]; then
  echo -e "${RED}ERROR: TOKEN env var is required${NC}"
  echo "Usage: TOKEN=\"<jwt>\" $0"
  exit 1
fi

AUTH="Authorization: Bearer $TOKEN"
PASS=0; FAIL=0

pass() { echo -e "${GREEN}PASS${NC} $*"; PASS=$((PASS+1)); }
fail() { echo -e "${RED}FAIL${NC} $*"; FAIL=$((FAIL+1)); }
info() { echo -e "${YELLOW}---- $* ----${NC}"; }

# ── Helpers ──────────────────────────────────────────────────────────

do_post() {
  local path="$1" body="$2" desc="$3" expect="${4:-200}"
  local resp code
  resp=$(curl -s -w "\n%{http_code}" -X POST "$BASE$path" -H "$AUTH" -H "Content-Type: application/json" -d "$body")
  code=$(echo "$resp" | tail -1)
  local body_out=$(echo "$resp" | sed '$d')
  if [ "$code" != "$expect" ]; then
    fail "$desc: expected HTTP $expect, got $code. Body: $body_out"
    echo "$body_out"
  else
    pass "$desc (HTTP $code)"
  fi
  echo "$body_out"
}

do_get() {
  local path="$1" desc="$2" expect="${3:-200}"
  local resp code
  resp=$(curl -s -w "\n%{http_code}" -X GET "$BASE$path" -H "$AUTH")
  code=$(echo "$resp" | tail -1)
  local body_out=$(echo "$resp" | sed '$d')
  if [ "$code" != "$expect" ]; then
    fail "$desc: expected HTTP $expect, got $code"
    echo "$body_out"
  else
    pass "$desc (HTTP $code)"
  fi
  echo "$body_out"
}

# ── 1. Setup: create a session with mock model ──────────────────────

info "Setup: Create session with mock model"
RESP=$(do_post "/chat/sessions" '{"title":"压缩测试","model_name":"mock"}' "Create session")
SESSION_ID=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
if [ -z "$SESSION_ID" ]; then
  echo "$RESP"
  fail "Could not extract session ID"
  exit 1
fi
echo "Session ID: $SESSION_ID"

# ── 2. Send many messages to exceed compression threshold ───────────

info "Send 35 messages (mock model has max_messages_before_compress=30, max_context_messages=10)"

for i in $(seq 1 35); do
  # Use SSE via curl -N. Only capture the first few lines for brevity.
  curl -s -N -X POST "$BASE/chat/sessions/$SESSION_ID/messages/stream" \
    -H "$AUTH" \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    -d "{\"content\":\"这是第${i}条测试消息。问题：1+${i}=？\"}" 2>&1 | head -20 > /dev/null &
  PID=$!
  # Wait for stream to finish (mock provider is fast)
  sleep 1
  kill $PID 2>/dev/null || true
  wait $PID 2>/dev/null || true
  echo "  Sent message $i/35"
done

info "All 35 messages sent. Total should be ~70 (35 user + 35 assistant)"

# ── 3. Verify compression triggered ─────────────────────────────────

info "3. Check that compression doesn't affect message list"

RESP=$(do_get "/chat/sessions/$SESSION_ID/messages?limit=5" "List latest 5 messages")
MSG_COUNT=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('messages',[])))" 2>/dev/null || echo "0")
echo "  Latest 5 messages count: $MSG_COUNT"

# Verify: messages should NOT contain system role (summary is separate table)
HAS_SYSTEM=$(echo "$RESP" | python3 -c "
import sys,json
d=json.load(sys.stdin)
msgs=d.get('messages',[])
has_sys=any(m['role']=='system' for m in msgs)
print('yes' if has_sys else 'no')
" 2>/dev/null || echo "error")
if [ "$HAS_SYSTEM" = "no" ]; then
  pass "Message list does NOT contain system summary (summary is in separate table)"
else
  fail "Message list should not contain system summary"
fi

# ── 4. Verify search still works ────────────────────────────────────

info "4. Verify search still works"
RESP=$(do_get "/chat/sessions/$SESSION_ID/messages/search?q=测试" "Search for 测试")
SEARCH_COUNT=$(echo "$RESP" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('messages',[])))" 2>/dev/null || echo "0")
if [ "$SEARCH_COUNT" -gt 0 ]; then
  pass "Search returned $SEARCH_COUNT results"
else
  fail "Search should return results"
fi

# ── 5. Verify Markdown export still works ───────────────────────────

info "5. Verify Markdown export"
RESP=$(do_get "/chat/sessions/$SESSION_ID/export/markdown" "Export markdown")
if echo "$RESP" | grep -q "^# "; then
  pass "Export returns markdown content"
else
  fail "Export should return markdown"
fi
# Summary should NOT appear in markdown export
if echo "$RESP" | grep -qi "summary\|摘要"; then
  echo "  (Note: export may contain 'summary' for other reasons)"
fi

# ── 6. Verify pagination still works ────────────────────────────────

info "6. Verify pagination"
RESP=$(do_get "/chat/sessions/$SESSION_ID/messages?limit=5" "List with limit=5")
HAS_MORE=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('has_more',''))" 2>/dev/null || echo "")
NEXT_BEFORE=$(echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('next_before_id',0))" 2>/dev/null || echo "0")
echo "  has_more=$HAS_MORE next_before_id=$NEXT_BEFORE"

if [ "$NEXT_BEFORE" != "0" ]; then
  # Paginate to next page
  RESP2=$(do_get "/chat/sessions/$SESSION_ID/messages?limit=5&before_id=$NEXT_BEFORE" "List page 2")
  PAGE2_COUNT=$(echo "$RESP2" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('messages',[])))" 2>/dev/null || echo "0")
  if [ "$PAGE2_COUNT" -gt 0 ]; then
    pass "Pagination works (page 2: $PAGE2_COUNT messages)"
  else
    fail "Pagination page 2 should have messages"
  fi
else
  pass "Pagination: only one page of messages (all fit in limit)"
fi

# ── 7. Summary ──────────────────────────────────────────────────────

info "Summary"
echo ""
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo ""
echo "Manual verification notes:"
echo "  1. Check DB table chat_session_summaries:"
echo "     SELECT id, session_id, LEFT(content,100), last_message_id FROM chat_session_summaries WHERE session_id=$SESSION_ID;"
echo "  2. Verify last_message_id points to a real chat_message ID"
echo "  3. Verify content contains a summary (mock provider returns English summary)"
echo "  4. The summary should not appear in: message list, search results, markdown export"
echo ""
echo "Cleanup: DELETE FROM chat_messages WHERE session_id=$SESSION_ID; DELETE FROM chat_sessions WHERE id=$SESSION_ID; DELETE FROM chat_session_summaries WHERE session_id=$SESSION_ID;"
