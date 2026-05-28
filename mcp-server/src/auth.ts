import * as crypto from "crypto";

export interface AuthContext {
  buyerId: string;
  entitlementId: string;
  agentId: string;
  scopes: string[];
  /** The short-lived entitlement token for backend calls */
  entitlementToken: string;
}

const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";
const JWT_SECRET = process.env.JWT_SECRET || "anyagent-dev-secret-change-in-production";

// Minimal JWT encoder for minting entitlement tokens locally.
// The token MUST contain scope=use so that RequireScope("use") passes in the Go backend.
function mintLocalEntitlementToken(
  buyerId: string,
  entitlementId: string,
  agentId: string
): string {
  const header = Buffer.from(JSON.stringify({ alg: "HS256", typ: "JWT" })).toString("base64url");
  const now = Math.floor(Date.now() / 1000);
  const payload = Buffer.from(
    JSON.stringify({
      sub: buyerId,
      entitlement_id: entitlementId,
      agent_id: agentId,
      iat: now,
      exp: now + 3600, // 1 hour validity; the backend re-checks quota on each call
      scopes: ["use"],
    })
  ).toString("base64url");
  const sig = crypto
    .createHmac("sha256", JWT_SECRET)
    .update(`${header}.${payload}`)
    .digest("base64url");
  return `${header}.${payload}.${sig}`;
}

// Validate an entitlement token with the backend.
// The token is the short-lived entitlement token minted by POST /entitlements/{id}/token.
export async function validateEntitlementToken(
  token: string,
  backendUrl: string
): Promise<AuthContext> {
  // Decode the JWT payload (without signature verification — the backend verifies it).
  // In production, also verify the signature.
  const payload = decodeJwtPayload(token);
  if (!payload) {
    throw new Error("Invalid token format");
  }

  // Check expiration
  const exp = payload.exp as number | undefined;
  if (exp && Date.now() / 1000 > exp) {
    throw new Error("Token expired");
  }

  // Verify with the backend (fresh quota state)
  let allowed = false;
  try {
    const res = await fetch(
      `${backendUrl}/api/v1/entitlements/check?agent_id=${payload.agent_id}&entitlement_id=${payload.entitlement_id}`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }
    );
    if (res.ok) {
      const data = (await res.json()) as { allowed: boolean; reason?: string };
      allowed = data.allowed ?? false;
    }
  } catch {
    // Network error — fall back to local validation for resilience
    const fallbackScopes = payload.scopes as string[] | undefined;
    allowed = fallbackScopes?.includes("use") ?? false;
  }

  if (!allowed) {
    throw new Error("Entitlement not allowed or quota exhausted");
  }

  return {
    buyerId: payload.sub as string,
    entitlementId: payload.entitlement_id as string,
    agentId: payload.agent_id as string,
    scopes: (payload.scopes as string[]) ?? [],
    entitlementToken: token,
  };
}

// validateBuyerToken validates a buyer's long-lived auth token and returns their info.
// This is used when the MCP client connects with the buyer's long-lived login token
// (not an entitlement token). The TS gateway then mints a short-lived entitlement
// token locally for backend calls.
export async function validateBuyerToken(
  token: string,
  backendUrl: string
): Promise<{ buyerId: string; entitlementId: string; agentId: string }> {
  const payload = decodeJwtPayload(token);
  if (!payload) {
    throw new Error("Invalid token format");
  }

  // Check the buyer token against the backend's auth/me
  try {
    const res = await fetch(`${backendUrl}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) {
      throw new Error(`auth/me returned ${res.status}`);
    }
    const user = (await res.json()) as { id: string };
    const buyerId = user.id;

    // The MCP client passes the entitlement_id in a custom header or query param.
    // For now, we require the token to be a minted entitlement token, not a raw buyer token.
    // The buyer obtains entitlement tokens via the web flow: POST /entitlements then /mint-token.
    throw new Error("MCP client must use an entitlement token from the /entitlements/{id}/token endpoint");
  } catch (err) {
    if (err instanceof Error && err.message.includes("entitlement token")) {
      throw err;
    }
    throw new Error(`Buyer token validation failed: ${err}`);
  }
}

// decodeJwtPayload decodes the middle segment of a JWT without verifying the signature.
function decodeJwtPayload(token: string): Record<string, unknown> | null {
  const parts = token.split(".");
  if (parts.length !== 3) return null;
  try {
    const payload = Buffer.from(parts[1], "base64url").toString("utf-8");
    return JSON.parse(payload) as Record<string, unknown>;
  } catch {
    return null;
  }
}
