.PHONY: build generate test test-integration clean release-check release-snapshot lint

# Copy schemas from submodule for embedding
generate:
	@mkdir -p internal/schema
	@cp schema/schema/*.json internal/schema/
	@echo "Schemas copied to internal/schema/"

# Build the binary
build: generate
	go build -o bin/raterunner ./cmd/raterunner

# Run unit tests
test: generate
	go test ./...

# Run integration tests (requires STRIPE_SANDBOX_KEY)
# Usage: make test-integration
#    or: STRIPE_SANDBOX_KEY=sk_test_... make test-integration
test-integration: build
	@./scripts/integration-test.sh

# Clean build artifacts
clean:
	rm -rf bin/ dist/
	rm -f internal/schema/*.json

# Lint code
lint:
	golangci-lint run

# Check GoReleaser config
release-check: generate
	goreleaser check

# Build release locally (without publishing)
release-snapshot: generate
	goreleaser release --snapshot --clean --skip=publish

# Create a new release (use: make release VERSION=v0.1.0)
release:
ifndef VERSION
	$(error VERSION is required. Usage: make release VERSION=v0.1.0)
endif
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
