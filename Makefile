.PHONY: build generate test test-integration clean

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
	rm -rf bin/
	rm -f internal/schema/*.json
