up:
	docker compose -f deployments/docker-compose.yml up -d

down:
	docker compose -f deployments/docker-compose.yml down

run:
	go run ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy
