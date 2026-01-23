.PHONY: build generate test clean

# Copy schemas from submodule for embedding
generate:
	@mkdir -p internal/schema
	@cp schema/schema/*.json internal/schema/
	@echo "Schemas copied to internal/schema/"

# Build the binary
build: generate
	go build -o bin/raterunner ./cmd/raterunner

# Run tests
test: generate
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f internal/schema/*.json
