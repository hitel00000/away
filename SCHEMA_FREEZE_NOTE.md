# V1 Event Schema Freeze Note

As part of E-003, the following v1 event and command schemas have been formally frozen and documented via golden JSON fixtures:

1. `message.created` (inbound channel/public message echo)
2. `dm.created` (inbound private message/DM echo)
3. `send_message` (outbound client-to-server command)

The `sync.snapshot` schema is not yet implemented/emitted in v1 and is therefore not frozen here.

## Inconsistencies Frozen (No Protocol Migration)

Per the rules of E-003, we avoided protocol redesigns and minimized the patch by keeping the current schemas exactly as they are. The following known schema inconsistencies remain and are now formally asserted by strict parsing tests:

- **Target / Buffer ID Inconsistency**: `message.created` uses `buffer_id` (e.g. `"chan:#test"`), but `send_message` uses `target` (e.g. `"#test"`). 
- **DM vs Channel Format Drift**: `dm.created` uses `peer` rather than `buffer_id` and omits the sender's `nick` field entirely. The client currently defaults missing sender `nick` values to `"me"` on display. This shape is preserved strictly as-is to avoid breaking or migrating the current web client.

## Artifacts Added
- `fixtures/events_v1/*.json`: Golden JSON examples.
- `relayd/schema_test.go`: Strict regression tests validating structs against golden fixtures without allowing unknown fields.
