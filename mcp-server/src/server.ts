import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { tools } from "./tools/index.js";

export function createServer(): McpServer {
  const server = new McpServer({
    name: "anyagent",
    version: "0.1.0",
  });

  // Register all tools
  for (const tool of tools) {
    server.tool(
      tool.name,
      tool.description,
      tool.inputSchema,
      tool.handler
    );
  }

  return server;
}
