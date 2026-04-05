start:
	COMPOSE_PROJECT_NAME=gratheon docker compose -f docker-compose.dev.yml up --build --renew-anon-volumes
stop:
	COMPOSE_PROJECT_NAME=gratheon docker compose -f docker-compose.dev.yml stop
run:
	go run .
test:
	go test ./...
