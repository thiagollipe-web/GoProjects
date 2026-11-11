.PHONY: build test vet run
build:
	go build -o bin/gateway ./cmd/gateway
test:
	go test -race ./...
vet:
	go vet ./...
run:
	go run ./cmd/gateway -config config.json
