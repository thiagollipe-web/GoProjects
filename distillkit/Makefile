.PHONY: build test vet run
build:
	CGO_ENABLED=0 go build -o bin/distillkit ./cmd/distillkit
test:
	go test -race ./...
vet:
	go vet ./...
run:
	go run ./cmd/distillkit
