import http from "node:http";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { createServer } from "../server.js";
import { validateEntitlementToken, type AuthContext } from "../auth.js";

const PORT = parseInt(process.env.MCP_HTTP_PORT || "3001", 10);
const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

// Mint a short-lived entitlement token for backend calls.
// The TS gateway calls the Go backend's entitlement token endpoint to get
// a scoped token (scope=use, entitlement_id, agent_id embedded) so that
// RunAgent's RequireScope("use") passes.
async function mintEntitlementToken(
  buyerToken: string,
  entitlementId: string
): Promise<string> {
  const resp = await fetch(`${BACKEND_URL}/api/v1/entitlements/${entitlementId}/token`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${buyerToken}`,
      "Content-Type": "application/json",
    },
  });
  if (!resp.ok) {
    throw new Error(`Failed to mint entitlement token: ${resp.status}`);
  }
  const data = (await resp.json()) as { token: string };
  return data.token;
}

export async function startHttpServer(): Promise<void> {
  const transport = new StreamableHTTPServerTransport({
    sessionIdGenerator: undefined, // stateless mode
  });

  const server = http.createServer(async (req, res) => {
    try {
      const authHeader =
        req.headers["authorization"] || req.headers["Authorization"];

      if (!authHeader || !Array.isArray(authHeader) && !authHeader.startsWith("Bearer ")) {
        res.writeHead(401, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ error: "Missing Authorization header" }));
        return;
      }

      const token = Array.isArray(authHeader)
        ? authHeader[0].slice("Bearer ".length)
        : authHeader.slice("Bearer ".length);

      let authCtx: AuthContext;
      try {
        authCtx = await validateEntitlementToken(token, BACKEND_URL);
      } catch {
        res.writeHead(401, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ error: "Invalid or expired entitlement token" }));
        return;
      }

      // Read request body for POST requests
      let body: unknown;
      if (req.method === "POST" || req.method === "GET") {
        body = await readBody(req);
      }

      // Create scoped server for this entitlement
      const scopedServer = createServer(authCtx);
      await scopedServer.connect(transport);

      await transport.handleRequest(
        req,
        res,
        body as Record<string, unknown> | undefined
      );
    } catch (err) {
      console.error("HTTP transport error:", err);
      if (!res.writableEnded) {
        res.writeHead(500, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ error: "Internal server error" }));
      }
    }
  });

  server.listen(PORT, () => {
    console.error(`AnyAgent MCP gateway started (HTTP on port ${PORT})`);
    console.error(
      `Connect: claude mcp add agentx -- agentx mcp --transport http --port ${PORT}`
    );
  });
}

function readBody(req: http.IncomingMessage): Promise<Record<string, unknown> | undefined> {
  return new Promise((resolve) => {
    if (req.method === "GET") {
      resolve(undefined);
      return;
    }
    let data = "";
    req.on("data", (chunk: Buffer) => {
      data += chunk.toString();
    });
    req.on("end", () => {
      if (!data) {
        resolve(undefined);
        return;
      }
      try {
        resolve(JSON.parse(data) as Record<string, unknown>);
      } catch {
        resolve(undefined);
      }
    });
    req.on("error", () => resolve(undefined));
  });
}
