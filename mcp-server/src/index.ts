#!/usr/bin/env node

import { startStdioServer } from "./transport/stdio.js";

async function main() {
  const args = process.argv.slice(2);

  if (args.includes("--help") || args.includes("-h")) {
    console.log(`
AnyAgent MCP Server

Usage:
  anyagent-mcp              Start MCP server (stdio transport)
  anyagent-mcp --help       Show this help

Connect with Claude Code:
  claude mcp add agentx -- anyagent-mcp
`);
    process.exit(0);
  }

  await startStdioServer();
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
