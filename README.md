# Mesh CLI

Command-line interface for Mesh — The Social Shell.

## Installation

```bash
npx @mndrk/mesh
```

Or install globally:

```bash
npm install -g @mndrk/mesh
```

## Usage

```bash
# Login with SSH key
mesh login

# Post something
mesh post "Hello, Mesh!"

# View your feed
mesh feed

# Follow someone
mesh follow @alice
```

See [joinme.sh](https://joinme.sh) for full documentation.

---

## Development

### Prerequisites

- Go 1.22+
- golangci-lint (optional)

### Setup

```bash
# 1. Install golangci-lint (optional)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 2. Build the CLI
make build

# 3. Run from source
make run
```

### Make Commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `bin/mesh` |
| `make run` | Run from source |
| `make test` | Run tests |
| `make test-cover` | Run tests with coverage |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code |
| `make clean` | Clean build artifacts |

### Project Structure

```
mesh-cli/
├── cmd/mesh/        # CLI entry point
├── pkg/
│   ├── api/        # Request/response types
│   ├── client/     # Backend API client
│   └── models/     # Shared data models
└── npm/            # NPM distribution wrapper
```

### Testing Against Local Backend

```bash
# Set API endpoint to local server
export MSH_API_URL=http://localhost:8080

# Or use config
mesh config set api_url http://localhost:8080
```

## License

MIT
