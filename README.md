# AnyAgent

> Enhance your Claude Code / Codex / Cursor with Agent Packs, project memory, traces, and team collaboration.

AnyAgent is an open-source agent enhancement layer. It lets you continue using your existing AI coding tools while adding persistent memory, execution traces, and reusable agent configurations.

## Why AnyAgent?

| Feature | Claude Code Skills | AnyAgent |
|---------|-------------------|----------|
| Project Memory | ❌ No persistence | ✅ Cross-session memory |
| Execution Traces | ❌ No tracing | ✅ Full trace with tool calls |
| Eval Rules | ❌ Manual | ✅ Automated evaluation |
| Team Sharing | ❌ Individual | ✅ Shared configs |
| Cross-tool | ❌ Locked to one | ✅ Claude/Codex/Cursor |

## Quick Start

```bash
# Install CLI
curl -fsSL https://raw.githubusercontent.com/anyagent/anyagent/main/scripts/install.sh | sh

# Initialize project
cd your-project
agentx init

# Install an agent
agentx install code-reviewer

# Start MCP server
agentx mcp

# Connect with Claude Code
claude mcp add agentx -- agentx mcp
```

## CLI Commands

```bash
# Project setup
agentx init                    # Initialize .agentx/ directory
agentx status                  # Show project status

# Agent management
agentx search <query>          # Search agent store
agentx install <name>          # Install agent pack
agentx list                    # List installed agents
agentx uninstall <name>        # Remove agent

# Memory
agentx memory add "..." --kind decision   # Add memory
agentx memory list             # List memories
agentx memory search "..."     # Search memories

# Traces
agentx trace list              # List execution traces
agentx trace show <id>         # Show trace details

# MCP Server
agentx mcp                     # Start MCP server (stdio)
```

## MCP Tools

When connected to Claude Code / Codex / Cursor via MCP:

| Tool | Description |
|------|-------------|
| `agentx_git_context` | Get git branch, status, diff |
| `agentx_read_file` | Read project files |
| `agentx_list_files` | List project files |
| `agentx_memory` | Search/save project memories |
| `agentx_run_command` | Run shell commands |

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│  Claude Code    │     │   Cursor        │
│  Codex          │     │   Other tools   │
└────────┬────────┘     └────────┬────────┘
         │ MCP                   │ MCP
         └───────────┬───────────┘
                     │
         ┌───────────▼───────────┐
         │    agentx CLI         │
         │  ┌─────────────────┐  │
         │  │   MCP Server    │  │
         │  │   Local Runner  │  │
         │  │   Memory        │  │
         │  │   Traces        │  │
         │  └─────────────────┘  │
         └───────────────────────┘
                     │
         ┌───────────▼───────────┐
         │    .agentx/           │
         │  ├── config.yaml      │
         │  ├── agents/          │
         │  ├── memory/          │
         │  └── traces/          │
         └───────────────────────┘
```

## Agent Pack Format

```yaml
# agent.yaml
name: code-reviewer
version: 0.1.0
description: Automated code review
category: coding
tags:
  - review
  - quality

prompts:
  - name: review
    path: prompts/review.md

tools:
  - name: git_diff
    description: Get git diff

eval:
  - name: has_comments
    path: eval/has_comments.yaml
```

## Project Structure

```
anyagent/
├── cli/              # Rust CLI (single binary)
├── mcp-server/       # TypeScript MCP Server
├── backend/          # Go API Server (optional)
├── web/              # Next.js Dashboard (optional)
├── agent-packs/      # Built-in agent packs
├── infra/            # Docker, deployment
└── scripts/          # Install, release scripts
```

## Development

```bash
# Build CLI
cd cli && cargo build --release

# Run backend (optional, needs PostgreSQL)
cd backend && go run ./cmd/server

# Run web dashboard (optional)
cd web && npm run dev

# Run MCP server in dev mode
cd mcp-server && npm run dev
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE)
