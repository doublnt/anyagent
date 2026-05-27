# Changelog

All notable changes to this project will be documented in this file.

## v0.1.0 (Initial Release)

### Features

- **CLI (agentx)**
  - `agentx init` - Initialize project with .agentx/ directory
  - `agentx login/logout` - Authentication
  - `agentx search` - Search Agent Store
  - `agentx install/uninstall` - Install agent packs
  - `agentx run` - Execute tasks with agents
  - `agentx mcp` - Start MCP server (stdio transport)
  - `agentx memory` - Manage project memories (local + cloud sync)
  - `agentx trace` - View and sync execution traces
  - `agentx config` - Configuration management

- **MCP Server**
  - 5 tools: git_context, read_file, list_files, memory, run_command
  - Stdio transport for Claude Code integration
  - npm package: @anyagent/mcp-server

- **Backend API**
  - JWT authentication
  - Agent registry (list, get, download)
  - Project management
  - Memory CRUD with search
  - Trace recording with spans
  - License/subscription management

- **Web Dashboard**
  - Agent Store with search and categories
  - Project dashboard
  - Memory management UI
  - Trace viewer
  - Login/Register

- **Infrastructure**
  - Docker Compose for local development
  - PostgreSQL with pgvector for memory search
  - Redis for caching

### Supported Platforms

- CLI: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- Backend: Docker (linux/amd64)
- Web: Any modern browser
