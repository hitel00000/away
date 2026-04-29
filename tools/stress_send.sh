#!/usr/bin/env bash
# E-001 stress test: verify client_id ordering under rapid sends.
#
# Requires: websocat (https://github.com/vi/websocat)
# Usage:    ./tools/stress_send.sh [WS_URL] [N]
#
# Expected: each own-message echo carries the client_id that was sent.
# Failure : any mismatch printed to stderr, exit code 1.

set -euo pipefail

WS_URL="${1:-ws://localhost:8080/ws}"
N="${2:-50}"
TARGET="#test"
TMPDIR_LOCAL=$(mktemp -d)
SENT="$TMPDIR_LOCAL/sent.txt"
RECV="$TMPDIR_LOCAL/recv.txt"

cleanup() { rm -rf "$TMPDIR_LOCAL"; }
trap cleanup EXIT

echo "stress_send: $N rapid sends → $WS_URL target=$TARGET"

# Build N send_message commands, one per line.
# Each carries a deterministic client_id: stress_NNN so we can verify order.
for i in $(seq -w 1 "$N"); do
    cid="stress_$(printf '%03d' "$i")"
    echo "$cid" >> "$SENT"
    printf '{"type":"send_message","payload":{"client_id":"%s","target":"%s","text":"msg %s"}}\n' \
        "$cid" "$TARGET" "$i"
done | websocat --no-close --text "$WS_URL" | while IFS= read -r line; do
    # Extract client_id from echoed event (message.created / dm.created).
    cid=$(echo "$line" | grep -o '"client_id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [ -n "$cid" ] && echo "$cid"
done | head -n "$N" > "$RECV"

echo "--- sent ---"
cat "$SENT"
echo "--- received ---"
cat "$RECV"

MISMATCHES=0
while IFS= read -r sent_id && IFS= read -r recv_id <&3; do
    if [ "$sent_id" != "$recv_id" ]; then
        echo "MISMATCH: sent=$sent_id  got=$recv_id" >&2
        MISMATCHES=$((MISMATCHES + 1))
    fi
done < "$SENT" 3< "$RECV"

if [ "$MISMATCHES" -eq 0 ]; then
    echo "OK: all $N client_ids reconciled in order."
    exit 0
else
    echo "FAIL: $MISMATCHES mismatches out of $N sends." >&2
    exit 1
fi
