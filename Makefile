.DEFAULT_GOAL := help
.PHONY: help test test-race test-integration test-coverage vet tidy modules-check generate check-generated security release-plan

help: ## Show available commands.
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run all unit tests and examples with exact coverage enforcement.
	@scripts/test-unit.sh

test-race: ## Run all modules and real example programs with race detection.
	@scripts/test-modules.sh race

test-integration: ## Run pinned ntfy, Gotify and Synapse servers using Docker.
	@scripts/test-integration.sh

test-coverage: test ## Produce the unit coverage profile and measured evidence.

vet: ## Vet every module.
	@scripts/test-modules.sh vet

tidy: ## Tidy every module while retaining development replacements.
	@scripts/test-modules.sh tidy

modules-check: ## Verify the complete module inventory.
	@GOWORK=off go -C docs run ./modules

generate: ## Generate README snippets, API links, and local SVG badges.
	@GOWORK=off go -C docs run ./readme

check-generated: ## Verify source parity and deterministic regeneration.
	@scripts/check-generated.sh

security: ## Scan every module with the pinned vulnerability scanner.
	@command -v govulncheck >/dev/null
	@while IFS= read -r module; do (cd "$$module" && GOWORK=off govulncheck ./...) || exit; done < scripts/module-manifest.txt
	@cd integration && GOWORK=off govulncheck -test -tags=integration ./...

release-plan: ## Plan independent runtime module tags, VERSION=v0.1.0.
	@scripts/release-plan.sh $(VERSION)
