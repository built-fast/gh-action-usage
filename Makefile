.PHONY: help
help: ## show this help text
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: test-shellcheck test-shfmt ## run full test suite

.PHONY: test-shellcheck
test-shellcheck: ## run shellcheck
	gh shellcheck

.PHONY: test-shfmt
test-shfmt: ## run shfmt
	shfmt -d -i 2 -sr -ci .

.PHONY: format
format: ## format all shell scripts
	shfmt -w -i 2 -sr -ci .
