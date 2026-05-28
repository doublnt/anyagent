#!/usr/bin/env node

import { startStdioServer } from "./transport/stdio.js";
import { startHttpServer } from "./transport/http.js";

async function main() {
  const args = process.argv.slice(2);

  if (args.includes("--help") || args.includes("-h")) {
    console.log(`
AnyAgent MCP Server

Usage:
  anyagent-mcp                      Start MCP server (stdio transport, default)
  anyagent-mcp --transport http     Start MCP server with HTTP/Streamable transport
  anyagent-mcp --port <port>       Port for HTTP transport (default: 3001)
  anyagent-mcp --help               Show this help

Stdio mode (default):
  anyagent-mcp
  claude mcp add agentx -- anyagent-mcp

HTTP mode (remote gateway):
  anyagent-mcp --transport http --port 3001
  claude mcp add agentx-remote -- agentx mcp --transport http --port 3001
`);
    process.exit(0);
  }

  const transport = args.includes("--transport") ? "http" : "stdio";
  const portIdx = args.indexOf("--port");
  if (portIdx !== -1 && args[portIdx + 1]) {
    process.env.MCP_HTTP_PORT = args[portIdx + 1];
  }

  if (transport === "http") {
    await startHttpServer();
  } else {
    await startStdioServer();
  }
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
