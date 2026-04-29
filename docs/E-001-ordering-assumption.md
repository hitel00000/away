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

**1. defer-in-loop fd leak**

`handler.go` had `defer f.Close()` inside the WebSocket read loop.
`defer` runs at function return, not block end. Rapid sends left
multiple FIFO fds open simultaneously, risking write interleaving.

Fixed: extracted `writeFifo()` helper; defer is scoped inside the
helper, so fd is released on each call return.

**2. Blocking O_WRONLY open**

`O_WRONLY` on a FIFO blocks until a reader is present. When irssi is
absent this stalls the WebSocket reader goroutine indefinitely, freezing
all message handling for that client.

Fixed: `O_WRONLY|syscall.O_NONBLOCK`. If no reader is present, open
returns `ENXIO` immediately. The error is logged; the WebSocket loop
continues.

**3. Error provenance**

Failures are now wrapped: `fmt.Errorf("open: %w", err)` and
`fmt.Errorf("write: %w", err)`. Log output identifies the failure site.

## Remaining risk

**FIFO partial-read (irssi plugin):**
`sysread` reads ≤ 4096 bytes. A line straddling a read boundary is
silently dropped → subsequent `shift @pending_ids` shifts the wrong id.
Low likelihood for typical message sizes. Deferred.

**No outbound queue in relay:**
When irssi is absent, commands are dropped (ENXIO). The client's 10 s
pending timeout surfaces the failure via the `unconfirmed` CSS class.
Accepted for MVP scope.

## Stress test

`tools/stress_send.sh [WS_URL] [N=50]`

Requires `websocat`. Sends N messages with deterministic client_ids
(`stress_001` … `stress_050`) and checks that echoed events return them
in send order.

## Conclusion

The ordering assumption is **valid** under normal conditions.
The relay no longer blocks on irssi absence.
Partial-read and command-drop risks are documented; not yet mitigated.
