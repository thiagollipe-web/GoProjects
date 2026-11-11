.PHONY: build test vet run
build:
	CGO_ENABLED=0 go build -o bin/reasoner ./cmd/reasoner
test:
	go test -race ./...
vet:
	go vet ./...
run:
	go run ./cmd/reasoner
