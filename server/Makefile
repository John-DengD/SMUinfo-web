.PHONY: sqlc build test run
sqlc:
	sqlc generate
build:
	CGO_ENABLED=0 go build -o bin/smudeal ./cmd/smudeal
test:
	go test ./...
run:
	go run ./cmd/smudeal
