PROJECTS := llm-gateway distillkit go-reasoner
BINS := llm-gateway:gateway distillkit:distillkit go-reasoner:reasoner

.DEFAULT_GOAL := help
.PHONY: help test vet fmt fmt-check build clean

help: ## list targets
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

test: ## run all tests with the race detector
	@for p in $(PROJECTS); do echo "== $$p"; (cd $$p && go test -race ./...) || exit 1; done

vet: ## go vet every project
	@for p in $(PROJECTS); do (cd $$p && go vet ./...) || exit 1; done

fmt: ## format all code
	@gofmt -w $(PROJECTS)

fmt-check: ## fail if any file is not formatted
	@test -z "$$(gofmt -l $(PROJECTS))" || (gofmt -l $(PROJECTS); exit 1)

build: ## build static binaries into ./bin
	@mkdir -p bin
	@for b in $(BINS); do p=$${b%%:*}; c=$${b##*:}; \
	  (cd $$p && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../bin/$$p ./cmd/$$c) || exit 1; done
	@ls -1 bin

clean: ## remove build output
	@rm -rf bin
