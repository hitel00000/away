# Away Architecture v0.3

Layers:
- irssi (source of truth)
- relay (coordination)
- web (projection)

Data flow:
irssi -> plugin -> relay -> browser

MVP:
- receive
- reply
- reconnect
