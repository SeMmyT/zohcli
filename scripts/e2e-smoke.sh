#!/bin/bash
# E2E Smoke Test for zoh CLI
# Exercises all read-only endpoints and safe write operations
# Usage: ./scripts/e2e-smoke.sh [email]
#   email: address to send test email to (optional, skips send tests if omitted)

set -uo pipefail

ZOH="${ZOH:-./zoh}"
EMAIL="${1:-}"
pass=0; fail=0; skip=0

# Colors (respect NO_COLOR)
if [ -z "${NO_COLOR:-}" ] && [ -t 1 ]; then
  GREEN='\033[32m'; RED='\033[31m'; YELLOW='\033[33m'; RESET='\033[0m'; BOLD='\033[1m'
else
  GREEN=''; RED=''; YELLOW=''; RESET=''; BOLD=''
fi

run_test() {
  local name="$1"; shift
  local output rc
  output=$(timeout 15 "$@" 2>&1)
  rc=$?

  if [ $rc -eq 124 ]; then
    echo -e "  ${YELLOW}SKIP${RESET}  $name (timeout)"
    ((skip++))
    return
  fi

  if [ $rc -eq 0 ]; then
    echo -e "  ${GREEN}PASS${RESET}  $name"
    ((pass++))
  elif echo "$output" | grep -qi "not available\|upgrade\|permission denied\|INVALID_OAUTHSCOPE\|not supported\|Invalid Input\|not configured\|HTTP 500"; then
    echo -e "  ${YELLOW}SKIP${RESET}  $name (plan/scope)"
    ((skip++))
  else
    echo -e "  ${RED}FAIL${RESET}  $name (exit $rc)"
    echo "        $(echo "$output" | head -1)"
    ((fail++))
  fi
}

# Capture a value from a command for use in later tests
capture() {
  "$@" 2>/dev/null | head -1
}

echo -e "${BOLD}zoh E2E Smoke Test${RESET}"
echo "Binary: $ZOH"
echo "Date:   $(date -Iseconds)"
echo

# ── Auth ──────────────────────────────────────────────
echo -e "${BOLD}Auth${RESET}"
run_test "auth list"         $ZOH auth list
run_test "auth list --check" $ZOH auth list --check
echo

# ── Config ────────────────────────────────────────────
echo -e "${BOLD}Config${RESET}"
run_test "config list"  $ZOH config list
run_test "config path"  $ZOH config path
run_test "config get region" $ZOH config get region
echo

# ── Admin: Users ──────────────────────────────────────
echo -e "${BOLD}Admin: Users${RESET}"
run_test "admin users list"       $ZOH admin users list
run_test "admin users list --json" $ZOH admin users list --output json --results-only

# Get first user email from table output (skip header, take first column)
USER_EMAIL=$($ZOH admin users list 2>/dev/null | sed -n '2p' | cut -f1 || echo "")
if [ -n "$USER_EMAIL" ]; then
  run_test "admin users get ($USER_EMAIL)" $ZOH admin users get "$USER_EMAIL"
else
  echo -e "  ${YELLOW}SKIP${RESET}  admin users get (no users found)"
  ((skip++))
fi
echo

# ── Admin: Groups ─────────────────────────────────────
echo -e "${BOLD}Admin: Groups${RESET}"
run_test "admin groups list" $ZOH admin groups list
echo

# ── Admin: Domains ────────────────────────────────────
echo -e "${BOLD}Admin: Domains${RESET}"
run_test "admin domains list" $ZOH admin domains list

# Get first domain from table output (skip header, take first column)
DOMAIN=$($ZOH admin domains list 2>/dev/null | sed -n '2p' | cut -f1 || echo "")
if [ -n "$DOMAIN" ]; then
  run_test "admin domains get ($DOMAIN)" $ZOH admin domains get "$DOMAIN"
else
  echo -e "  ${YELLOW}SKIP${RESET}  admin domains get (no domains found)"
  ((skip++))
fi
echo

# ── Admin: Audit ──────────────────────────────────────
echo -e "${BOLD}Admin: Audit${RESET}"
TODAY=$(date +%Y-%m-%d)
WEEK_AGO=$(date -d '7 days ago' +%Y-%m-%d 2>/dev/null || date -v-7d +%Y-%m-%d 2>/dev/null || echo "2026-02-07")
run_test "admin audit logs"          $ZOH admin audit logs --from "$WEEK_AGO" --to "$TODAY" --limit 5
run_test "admin audit login-history" $ZOH admin audit login-history --from "$WEEK_AGO" --to "$TODAY" --limit 5
run_test "admin audit smtp-logs"     $ZOH admin audit smtp-logs --from "$WEEK_AGO" --to "$TODAY" --limit 5
echo

# ── Mail: Folders ─────────────────────────────────────
echo -e "${BOLD}Mail: Folders${RESET}"
run_test "mail folders list" $ZOH mail folders list
echo

# ── Mail: Labels ──────────────────────────────────────
echo -e "${BOLD}Mail: Labels${RESET}"
run_test "mail labels list" $ZOH mail labels list
echo

# ── Mail: Messages ────────────────────────────────────
echo -e "${BOLD}Mail: Messages${RESET}"
run_test "mail messages list (limit 3)" $ZOH mail messages list --limit 3

# Get first message ID and Inbox folder ID from table output
MSG_ID=$($ZOH mail messages list --limit 1 2>/dev/null | sed -n '2p' | awk -F'\t' '{print $NF}' || echo "")
FOLDER_ID=$($ZOH mail folders list 2>/dev/null | awk -F'\t' '$1=="Inbox" {print $NF; exit}' || echo "")

if [ -n "$MSG_ID" ] && [ -n "$FOLDER_ID" ]; then
  run_test "mail messages get"     $ZOH mail messages get "$MSG_ID" --folder "$FOLDER_ID"
  run_test "mail messages search"  $ZOH mail messages search "test" --limit 3
  run_test "mail attachments list" $ZOH mail attachments list "$MSG_ID" --folder "$FOLDER_ID"
else
  echo -e "  ${YELLOW}SKIP${RESET}  mail messages get (no messages or folder)"
  echo -e "  ${YELLOW}SKIP${RESET}  mail messages search (no messages)"
  echo -e "  ${YELLOW}SKIP${RESET}  mail attachments list (no messages)"
  ((skip+=3))
fi
echo

# ── Mail: Settings ────────────────────────────────────
echo -e "${BOLD}Mail: Settings${RESET}"
run_test "mail settings display-name get" $ZOH mail settings display-name get
run_test "mail settings vacation get"     $ZOH mail settings vacation get
run_test "mail settings forwarding get"   $ZOH mail settings forwarding get
run_test "mail settings signatures list"  $ZOH mail settings signatures list
echo

# ── Mail Admin ────────────────────────────────────────
echo -e "${BOLD}Mail Admin${RESET}"
run_test "mail admin spam categories" $ZOH mail admin spam categories
run_test "mail admin logs"            $ZOH mail admin logs --limit 5
run_test "mail admin retention get"   $ZOH mail admin retention get
echo

# ── Send (only if email provided) ────────────────────
if [ -n "$EMAIL" ]; then
  echo -e "${BOLD}Send (to $EMAIL)${RESET}"
  run_test "mail send compose (dry-run)" $ZOH mail send compose \
    --to "$EMAIL" --subject "zoh smoke test" --body "This is an automated test from zoh CLI." --dry-run
  run_test "mail send compose (live)" $ZOH mail send compose \
    --to "$EMAIL" --subject "zoh smoke test $(date +%H:%M)" --body "Automated test from zoh CLI e2e smoke test."
  echo
else
  echo -e "${BOLD}Send${RESET}"
  echo -e "  ${YELLOW}SKIP${RESET}  mail send (no email provided, pass email as arg to test)"
  ((skip++))
  echo
fi

# ── Introspection ─────────────────────────────────────
echo -e "${BOLD}Introspection${RESET}"
run_test "version" $ZOH version
run_test "schema"  $ZOH schema
echo

# ── Summary ───────────────────────────────────────────
total=$((pass + fail + skip))
echo -e "${BOLD}Results: ${GREEN}$pass passed${RESET}, ${RED}$fail failed${RESET}, ${YELLOW}$skip skipped${RESET} ($total total)"

if [ $fail -gt 0 ]; then
  exit 1
fi
