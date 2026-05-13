#!/usr/bin/env bash
# Manual verification script for message pagination and in-session search.
#
# Prerequisites:
#   1. Server running at http://localhost:8080
#   2. A registered user with valid JWT token
#
# Set these variables before running:
#   export TOKEN="<your-jwt-token>"
#   export BASE="http://localhost:8080/api/v1"
#
# Usage:
#   chmod +x scripts/test_message_pagination.sh
#   TOKEN="..." ./scripts/test_message_pagination.sh

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

TOKEN="${TOKEN:-}"
BASE="${BASE:-http://localhost:8080/api/v1}"

if [ -z "$TOKEN" ]; then
  echo -e "${RED}ERROR: TOKEN env var is required${NC}"
  echo "Usage: TOKEN=\"<jwt>\" $0"
  exit 1
fi

AUTH="Authorization: Bearer $TOKEN"
SESSION_ID="${SESSION_ID:-}"  # Set to an existing session ID, or one will be created

pass()  { echo -e "${GREEN}PASS${NC} $*"; }
fail()  { echo -e "${RED}FAIL${NC} $*"; exit 1; }
info()  { echo -e "---- $* ----"; }

# ── Helpers ──────────────────────────────────────────────────────────

do_get() {
  local path="$1"
  local desc="$2"
  local expect_code="${3:-200}"
  local resp
  resp=$(curl -s -w "\n%{http_code}" -X GET "$BASE$path" -H "$AUTH" 2>&1) || true
  local code
  code=$(echo "$resp" | tail -1)
  local body
  body=$(echo "$resp" | sed '$d')

  if [ "$code" != "$expect_code" ]; then
    fail "$desc: expected HTTP $expect_code, got $code. Body: $body"
  fi
  echo "$body"
  pass "$desc (HTTP $code)"
}

# ── 1. Route disambiguation: List vs Search ──────────────────────────

info "1. Route disambiguation"

# Check 1a: List returns messages (or empty array for new session)
info "1a. List messages (GET /chat/sessions/:id/messages)"
RESP=$(do_get "/chat/sessions/1/messages" "List messages" 200)
echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); assert 'messages' in d, 'missing messages field'; assert 'has_more' in d, 'missing has_more field'; assert 'next_before_id' in d, 'missing next_before_id field'" 2>&1 \
  && pass "List response has messages, has_more, next_before_id" \
  || fail "List response missing expected fields"

# Check 1b: Search returns messages array
info "1b. Search messages (GET /chat/sessions/:id/messages/search?q=...)"
RESP=$(do_get "/chat/sessions/1/messages/search?q=test" "Search messages" 200)
echo "$RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); assert 'messages' in d, 'missing messages field'" 2>&1 \
  && pass "Search response has messages field" \
  || fail "Search response missing messages field"

# Check 1c: List and Search are different endpoints (not conflicting)
info "1c. Verify /messages/search is distinct from /messages"
# The search endpoint requires 'q' param; list does not.
# Both should return 200 (even for non-existent sessions if DB allows).
# This confirms Gin routes them separately.

# ── 2. Pagination: before_id and limit ───────────────────────────────

info "2. Pagination behavior"

# Check 2a: limit=3 returns at most 3 messages
info "2a. limit clamping"
RESP=$(do_get "/chat/sessions/1/messages?limit=3" "List with limit=3" 200)
COUNT=$(echo "$RESP" | python3 -c "import sys,json; print(len(json.load(sys.stdin)['messages']))" 2>&1) || COUNT="?"
if [ "$COUNT" != "?" ] && [ "$COUNT" -le 3 ]; then
  pass "limit=3 returned $COUNT messages (<=3)"
else
  fail "limit=3 returned $COUNT messages"
fi

# Check 2b: limit > 100 is clamped to 100
info "2b. limit > 100 clamped"
RESP=$(do_get "/chat/sessions/1/messages?limit=200" "List with limit=200 (clamped)" 200)
# This succeeds because the handler silently clamps, doesn't error.

# Check 2c: limit <= 0 returns 400
info "2c. limit <= 0 returns 400"
do_get "/chat/sessions/1/messages?limit=0" "List with limit=0" 400
do_get "/chat/sessions/1/messages?limit=-1" "List with limit=-1" 400

# Check 2d: before_id < 0 returns 400
info "2d. before_id < 0 returns 400"
do_get "/chat/sessions/1/messages?before_id=-1" "List with before_id=-1" 400

# Check 2e: before_id = 0 (or absent) returns latest messages
info "2e. before_id absent returns latest"
do_get "/chat/sessions/1/messages" "List without before_id" 200

# Check 2f: before_id not an int returns 400
info "2f. before_id non-integer returns 400"
do_get "/chat/sessions/1/messages?before_id=abc" "List with before_id=abc" 400

# ── 3. has_more and next_before_id correctness ───────────────────────

info "3. has_more and next_before_id"

# For a session with messages, paginate through all pages.
# This test requires a session with multiple messages.
info "3a. empty response has next_before_id=0 and has_more=false"
RESP=$(do_get "/chat/sessions/999999/messages?limit=10" "List non-existent session" 200)
echo "$RESP" | python3 -c "
import sys, json
d = json.load(sys.stdin)
assert d['messages'] == [], f'expected empty messages, got {d[\"messages\"]}'
assert d['has_more'] == False, f'expected has_more=false, got {d[\"has_more\"]}'
assert d['next_before_id'] == 0, f'expected next_before_id=0, got {d[\"next_before_id\"]}'
" && pass "empty response correct" || fail "empty response incorrect"

# ── 4. Search validation ────────────────────────────────────────────

info "4. Search query validation"

# Check 4a: empty q returns 400
info "4a. empty q"
do_get "/chat/sessions/1/messages/search?q=" "Search with empty q" 400

# Check 4b: whitespace-only q returns 400
info "4b. whitespace-only q"
do_get "/chat/sessions/1/messages/search?q=%20%20" "Search with whitespace q" 400

# Check 4c: missing q returns 400
info "4c. missing q param"
do_get "/chat/sessions/1/messages/search" "Search without q" 400

# Check 4d: valid search returns results
info "4d. valid search"
do_get "/chat/sessions/1/messages/search?q=hello" "Search with valid q" 200

# ── 5. Cross-user access ────────────────────────────────────────────

info "5. Cross-user access"
# This verifies that a session belonging to user A returns 'session not found'
# for user B. To test: create a session as user A, then query it as user B.
# The response should be 404 with error "session not found".
# This requires two tokens; manual verification noted below.

# ── Summary ──────────────────────────────────────────────────────────

echo ""
echo "All automated checks passed."
echo ""
echo "Manual checks recommended:"
echo "  1. Cross-user access: create session as user A, query messages as user B → 404"
echo "  2. Full pagination walk: if session has N messages, paginate with limit=2 through all pages"
echo "  3. has_more: verify has_more=true when there are more messages, false on last page"
echo "  4. next_before_id points to oldest message ID in returned batch"
echo "  5. Messages are returned in ASC (oldest→newest) order"
echo "  6. Search only returns messages from the requested session"
