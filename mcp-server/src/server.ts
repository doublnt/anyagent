import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { tools } from "./tools/index.js";
import { AuthContext } from "./auth.js";
import { buildHostedAgentTool } from "./tools/hosted-agent.js";

export { AuthContext };

// createServer creates an MCP server. If authCtx is provided (HTTP gateway mode),
// it registers the dynamic hosted-agent tool scoped to that entitlement.
export function createServer(authCtx?: AuthContext): McpServer {
  const server = new McpServer({
    name: "anyagent",
    version: "0.1.0",
  });

  // Register local tools (always available in stdio mode)
  for (const tool of tools) {
    server.tool(tool.name, tool.description, tool.inputSchema, tool.handler);
  }

  // In HTTP/gateway mode, also register the hosted agent tool for this entitlement
  if (authCtx) {
    const { definition, handler } = buildHostedAgentTool({
      authCtx,
      agentName: "agent", // The MCP gateway routes by agent; this is the default tool name
    });
    server.tool(definition.name, definition.description, definition.inputSchema, handler);
  }

  return server;
}
