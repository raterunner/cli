# Raterunner CLI

A command-line tool for managing SaaS billing configurations as code. Define your pricing plans, entitlements, and promotions in YAML, then sync them to payment providers.

## Why Raterunner?

Managing billing across SaaS applications is painful:

- **Scattered configuration** — pricing lives in Stripe dashboard, entitlements in code, feature flags elsewhere
- **No version control** — changes to plans are hard to track, review, or rollback
- **Environment drift** — sandbox and production get out of sync
- **Provider lock-in** — switching from Stripe to Paddle means rewriting everything

Raterunner solves this with a **single source of truth**:

```yaml
plans:
  - id: pro
    name: Pro Plan
    prices:
      monthly: { amount: 2900 }
      yearly: { amount: 29000 }
    limits:
      seats: 10
      api_calls: 50000
    features:
      - "Up to 10 users"
      - "Priority support"
```

**Benefits:**

- **Version control** — track all billing changes in Git
- **Code review** — pricing changes go through PR review like any other code
- **Environment parity** — same config deploys to sandbox and production
- **Schema validation** — catch errors before they hit production
- **CI/CD ready** — automate billing deployments

## Current Status

> **Note:** Raterunner currently supports **Stripe** only. Support for Paddle and Chargebee is planned.
>
> If you need support for other providers, please contact [raterunner@akorchak.software](mailto:raterunner@akorchak.software).

## Installation

### Homebrew (macOS/Linux)

```bash
brew install raterunner/tap/raterunner
```

### Download binary

Download the latest release from [GitHub Releases](https://github.com/raterunner/cli/releases):

```bash
# macOS (Apple Silicon)
curl -Lo raterunner.tar.gz https://github.com/raterunner/cli/releases/latest/download/raterunner_Darwin_arm64.tar.gz
tar -xzf raterunner.tar.gz
sudo mv raterunner /usr/local/bin/

# macOS (Intel)
curl -Lo raterunner.tar.gz https://github.com/raterunner/cli/releases/latest/download/raterunner_Darwin_amd64.tar.gz
tar -xzf raterunner.tar.gz
sudo mv raterunner /usr/local/bin/

# Linux (x86_64)
curl -Lo raterunner.tar.gz https://github.com/raterunner/cli/releases/latest/download/raterunner_Linux_amd64.tar.gz
tar -xzf raterunner.tar.gz
sudo mv raterunner /usr/local/bin/

# Linux (ARM64)
curl -Lo raterunner.tar.gz https://github.com/raterunner/cli/releases/latest/download/raterunner_Linux_arm64.tar.gz
tar -xzf raterunner.tar.gz
sudo mv raterunner /usr/local/bin/
```

### Linux packages

```bash
# Debian/Ubuntu
curl -Lo raterunner.deb https://github.com/raterunner/cli/releases/latest/download/raterunner_amd64.deb
sudo dpkg -i raterunner.deb

# RHEL/Fedora
curl -Lo raterunner.rpm https://github.com/raterunner/cli/releases/latest/download/raterunner_amd64.rpm
sudo rpm -i raterunner.rpm

# Alpine
curl -Lo raterunner.apk https://github.com/raterunner/cli/releases/latest/download/raterunner_amd64.apk
sudo apk add --allow-untrusted raterunner.apk
```

### Go install

```bash
go install github.com/raterunner/cli/cmd/raterunner@latest
```

### From source

```bash
git clone https://github.com/raterunner/cli.git
cd cli
make build
sudo mv bin/raterunner /usr/local/bin/
```

### Requirements

- Stripe API keys (for sync operations)

## Quick Start

### 1. Create a billing configuration

```yaml
# billing.yaml
version: 1
providers:
  - stripe

settings:
  currency: usd
  trial_days: 14

entitlements:
  seats:
    type: int
    unit: seat
  api_calls:
    type: int
    unit: call

plans:
  - id: free
    name: Free Plan
    prices:
      monthly: { amount: 0 }
    limits:
      seats: 1
      api_calls: 1000

  - id: pro
    name: Pro Plan
    trial_days: 14
    prices:
      monthly: { amount: 2900 }
      yearly: { amount: 29000 }
    limits:
      seats: 10
      api_calls: 50000
```

### 2. Validate the configuration

```bash
raterunner validate billing.yaml
```

### 3. Preview changes (dry run)

```bash
export STRIPE_SANDBOX_KEY=sk_test_...
raterunner apply --env sandbox --dry-run billing.yaml
```

### 4. Apply to Stripe

```bash
raterunner apply --env sandbox billing.yaml
```

## Commands

### `validate`

Validate a billing configuration against the JSON Schema.

```bash
raterunner validate billing.yaml
raterunner validate provider_stripe.yaml
```

### `apply`

Sync billing configuration to Stripe.

```bash
# Preview changes
raterunner apply --env sandbox --dry-run billing.yaml

# Apply changes
raterunner apply --env sandbox billing.yaml
raterunner apply --env production billing.yaml

# Output diff as JSON
raterunner apply --env sandbox --dry-run --json billing.yaml
```

### `import`

Import existing Stripe products/prices to a YAML file.

```bash
raterunner import --env sandbox --output imported.yaml
```

### `truncate`

Archive all products and prices in Stripe sandbox (useful for testing).

```bash
raterunner truncate           # Interactive confirmation
raterunner truncate --confirm # Skip confirmation (for CI/CD)
```

### `config`

Manage CLI settings.

```bash
raterunner config set quiet true   # Enable quiet mode permanently
raterunner config get quiet        # Get current value
raterunner config list             # List all settings
raterunner config path             # Show config file path
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--quiet`, `-q` | Suppress non-essential output (errors still shown) |
| `--help`, `-h` | Show help |
| `--version`, `-v` | Show version |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `STRIPE_SANDBOX_KEY` | Stripe test API key (`sk_test_...`) |
| `STRIPE_PRODUCTION_KEY` | Stripe live API key (`sk_live_...`) |

## Configuration Schema

The billing configuration schema is maintained in a separate repository:

**[github.com/raterunner/schema](https://github.com/raterunner/schema)**

### Supported Features

| Feature | Status |
|---------|--------|
| Flat pricing | Supported |
| Per-unit pricing | Supported |
| Tiered pricing (graduated/volume) | Supported |
| Trial periods | Supported |
| Addons | Supported |
| Promotions/Coupons | Supported |
| Marketing features | Supported |
| Custom metadata | Supported |
| Multi-currency | Planned |

### Schema Files

- `billing.schema.json` — main configuration (plans, entitlements, addons, promotions)
- `provider.schema.json` — provider-specific mappings

## Project Structure

```
cmd/raterunner/           # CLI entrypoint
  main.go                 # Commands and flags
  main_test.go            # Integration tests
  testdata/               # Test fixtures

internal/
  config/                 # Configuration types and loading
  stripe/                 # Stripe API client
  diff/                   # Comparison and output
  validator/              # JSON Schema validation
  schema/                 # Embedded JSON schemas
```

## Development

```bash
# Build
make build

# Run tests
make test

# Update schemas from submodule
git submodule update --remote schema
make generate
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License — see [LICENSE](LICENSE) for details.

## Author

**Andrey Korchak**
Email: [raterunner@akorchak.software](mailto:raterunner@akorchak.software)
