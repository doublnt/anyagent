# Contributing to AnyAgent

Thanks for your interest in contributing! This guide will help you get started.

## Ways to Contribute

- **Report bugs** - Open an issue with steps to reproduce
- **Suggest features** - Open an issue with your use case
- **Submit code** - Fork, branch, and PR
- **Improve docs** - Fix typos, add examples
- **Create agent packs** - Share your agent configurations

## Development Setup

### Prerequisites

- Rust 1.70+ (for CLI)
- Go 1.22+ (for backend)
- Node.js 20+ (for web and MCP server)
- PostgreSQL 16+ with pgvector (optional, for backend)

### Getting Started

```bash
# Clone the repo
git clone https://github.com/anyagent/anyagent.git
cd anyagent

# Build CLI
cd cli
cargo build
cargo test

# Build backend
cd ../backend
go build ./...
go test ./...

# Build MCP server
cd ../mcp-server
npm install
npm run build

# Build web
cd ../web
npm install
npm run build
```

## Code Structure

```
anyagent/
├── cli/src/
│   ├── main.rs           # CLI entry point
│   ├── commands/         # Command implementations
│   ├── api/              # API client types
│   ├── config/           # Configuration
│   ├── runner/           # Local execution
│   └── mcp/              # MCP server (Rust)
├── mcp-server/src/
│   ├── index.ts          # Entry point
│   ├── server.ts         # MCP server setup
│   └── tools/            # Tool implementations
├── backend/
│   ├── cmd/server/       # HTTP server
│   ├── internal/
│   │   ├── handler/      # HTTP handlers
│   │   ├── db/           # Database layer
│   │   └── middleware/    # Auth, logging
│   └── migrations/       # SQL migrations
└── web/src/
    ├── app/              # Next.js pages
    └── lib/              # Shared utilities
```

## Submitting Changes

1. Fork the repository
2. Create a branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Add tests if applicable
5. Run tests: `cargo test` / `go test ./...` / `npm test`
6. Commit with a clear message
7. Push and open a PR

## Commit Messages

Use clear, descriptive commits:

```
feat: add memory search command
fix: handle empty git status
docs: update CLI usage examples
refactor: simplify API client
```

## Adding Agent Packs

1. Create a directory in `agent-packs/`
2. Add `agent.yaml` with metadata
3. Add prompts, tools, eval rules
4. Submit a PR

Example:
```
agent-packs/my-agent/
├── agent.yaml
├── prompts/
│   └── main.md
└── eval/
    └── quality.yaml
```

## Code Style

- **Rust**: Follow `cargo clippy` suggestions
- **Go**: Use `gofmt` and `go vet`
- **TypeScript**: Use ESLint defaults

## Questions?

Open an issue for discussion.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
