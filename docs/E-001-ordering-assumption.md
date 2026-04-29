# E-001: Own-Message Ordering Assumption

## Status
Documented. Bug fixed. Test script added.

## Assumption

Irssi `own_public` / `own_private` signals fire in the same order as
`/msg` commands were issued by `poll_commands`.

This is the foundation for `@pending_ids` FIFO correlation (D-003a).

## Why it holds

`poll_commands` runs in Irssi's single-threaded event loop, 250 ms timer.

Within a single `sysread` call, lines are processed in buffer order:

```
for my $line (split /\n/, $buf) {
    push @pending_ids, $client_id;   # 1. enqueue id
    $servers[0]->command("msg ...");  # 2. issue command (synchronous)
}
```

Irssi processes `/msg` commands synchronously within the same event loop
iteration. The resulting `own_public` signal fires before the next
`/msg` is issued. Therefore shift-order of `@pending_ids` matches
push-order.

## Risk: sysread partial line

`sysread` reads up to 4096 bytes. If a line straddles a read boundary
it will be silently dropped (current implementation has no line buffer).

Impact: dropped command → no pending_id pushed → subsequent reconcile
shifts the wrong id.

Likelihood: low for normal use (commands << 4096 bytes).

Accepted debt: noted in D-001 docs. Fix is a proper line-buffer in
`poll_commands`, deferred until needed.

## Fix applied (this task)

`handler.go` had `defer f.Close()` inside the WebSocket read loop.
`defer` runs at function return, not at block end. Rapid sends left
multiple FIFO fds open simultaneously, which could cause write
interleaving on the FIFO.

Fixed: extracted `writeFifo()` helper that opens, writes, and closes
the FIFO atomically per message. `defer` is now scoped inside the helper.

## Stress test

`tools/stress_send.sh [WS_URL] [N=50]`

Requires `websocat`. Sends N messages with deterministic client_ids
(`stress_001` … `stress_050`) and checks that the echoed events return
them in the same order.

## Conclusion

The ordering assumption is **valid** under normal conditions.
The FIFO partial-read risk is documented but not yet mitigated.
