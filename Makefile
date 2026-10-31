up:
	docker compose up -d --wait
down:
	docker compose down
integration:
	DB_DSN="postgres://student:student@localhost:15446/notes?sslmode=disable" go test -count=1 -v ./integration/...
test:
	go test ./...
