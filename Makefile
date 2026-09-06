.PHONY: test test-unit test-integration run migrate mongo-up mongo-test-up

test:
	go test ./...

test-unit:
	go test ./internal/...

test-integration:
	MONGODB_TEST_URI=mongodb://localhost:27018 go test ./tests/integration/...

run:
	go run ./cmd/api

migrate:
	go run ./cmd/migrate

mongo-up:
	docker compose up -d mongodb

mongo-test-up:
	docker compose -f docker-compose.test.yml up -d
