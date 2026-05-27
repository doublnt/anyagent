# @anyagent/mcp-server

MCP Server for AnyAgent - integrates with Claude Code, Codex, and Cursor.

## Installation

```bash
npm install -g @anyagent/mcp-server
```

## Usage

### With Claude Code

```bash
claude mcp add anyagent -- anyagent-mcp
```

### With Cursor

Add to your MCP settings:

```json
{
  "mcpServers": {
    "anyagent": {
      "command": "anyagent-mcp"
    }
  }
}
```

## Tools

The server exposes these tools:

- `agentx_git_context` - Get git context (branch, status, diff)
- `agentx_read_file` - Read files from the project
- `agentx_list_files` - List project files
- `agentx_memory` - Search/save project memories
- `agentx_run_command` - Run shell commands

## Development

```bash
npm install
npm run dev
```

## License

MIT
