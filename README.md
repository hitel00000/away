# Away

Personal ambient companion for an existing irssi workflow.

Not another IRC client.

Persistent IRC presence,
without persistent attention.

## What is Away?
Away is a lightweight mobile-first layer over irssi.

Think: ambient inbox for IRC.

## Architecture
irssi -> Away Plugin -> Relay Daemon -> Web Client

## MVP Scope
- receive messages
- send replies
- survive reconnects

## Repository Layout
docs/
irssi-plugin/
relayd/
web/
schemas/

See CONTEXT.md, AGENTS.md, docs/architecture.md
