run-relay:
	go run ./relayd/cmd/awayd

run-web:
	cd web && echo "placeholder web dev server"

test:
	go test ./...
