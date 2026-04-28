run-relay:
	go run ./relayd/cmd/awayd

run-web:
	cd web && echo "placeholder web dev server"

dev-feed:
	go run ./relayd/cmd/dev-feed -fixture fixtures/simple_chat.ndjson

test:
	go test ./...
