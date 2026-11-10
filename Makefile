BINARY := gh-action-usage

.PHONY: help
help: ## show this help text
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## build the extension binary
	go build -o $(BINARY) .

.PHONY: install
install: build ## build and install the extension locally
	gh extension install .

.PHONY: test
test: vet ## run full test suite
	go test ./...

.PHONY: vet
vet: ## run go vet and check formatting
	go vet ./...
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "run 'make format'" && exit 1)

.PHONY: format
format: ## format Go sources
	gofmt -w .

.PHONY: clean
clean: ## remove build artifacts
	rm -f $(BINARY)
