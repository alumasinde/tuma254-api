test:
	go test ./...

migrate:
	go run ./cmd/migrate up

run:
	go run ./cmd/api
