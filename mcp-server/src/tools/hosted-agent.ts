import { z } from "zod";
import { AuthContext } from "../auth.js";

const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

export interface HostedAgentToolOptions {
  authCtx: AuthContext;
  agentName: string; // e.g. "code-reviewer"
}

// buildHostedAgentTool returns a tool definition and handler for the hosted agent.
export function buildHostedAgentTool(options: HostedAgentToolOptions) {
  const { authCtx, agentName } = options;
  const toolName = `${agentName.replace(/-/g, "_")}__call`;

  const definition = {
    name: toolName,
    description: `Call the hosted agent "${agentName}" with your structured input`,
    inputSchema: {
      task: z.string().describe("The task description for the agent"),
      diff: z.string().optional().describe("Git diff or code changes to review/analyze"),
      files: z.array(z.string()).optional().describe("File paths relevant to the task"),
      context: z.string().optional().describe("Additional context (optional)"),
    },
  };

  const handler = async (args: {
    task: string;
    diff?: string;
    files?: string[];
    context?: string;
  }) => {
    const input = JSON.stringify({
      task: args.task ?? "",
      diff: args.diff ?? "",
      files: args.files ?? [],
      context: args.context ?? "",
    });

    // Call the Go backend's run endpoint.
    // Use the short-lived entitlement token so RequireScope("use") passes.
    const url = `${BACKEND_URL}/api/v1/agents/${agentName}/run`;
    let response: Response;
    try {
      response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${authCtx.entitlementToken}`,
        },
        body: JSON.stringify({
          input,
          entitlement_id: authCtx.entitlementId,
        }),
        signal: AbortSignal.timeout(120_000), // 2 min timeout for sandbox run
      });
    } catch (err) {
      return {
        content: [
          {
            type: "text" as const,
            text: JSON.stringify({ error: `Network error: ${err}` }),
          },
        ],
        isError: true,
      };
    }

    if (!response.ok) {
      let errorBody = "";
      try {
        errorBody = (await response.text()).slice(0, 200);
      } catch {}

      if (response.status === 403) {
        return {
          content: [
            {
              type: "text" as const,
              text: JSON.stringify({
                error: "quota_exceeded",
                message:
                  "Your subscription quota has been exhausted for this period.",
              }),
            },
          ],
          isError: true,
        };
      }

      return {
        content: [
          {
            type: "text" as const,
            text: JSON.stringify({
              error: `Agent call failed (${response.status})`,
              detail: errorBody,
            }),
          },
        ],
        isError: true,
      };
    }

    interface RunResponse {
      result: string;
      input_tokens: number;
      output_tokens: number;
      cost_micros: number;
      trace_id: string;
    }

    let runResult: RunResponse;
    try {
      runResult = (await response.json()) as RunResponse;
    } catch {
      return {
        content: [
          {
            type: "text" as const,
            text: JSON.stringify({ error: "Invalid response from backend" }),
          },
        ],
        isError: true,
      };
    }

    // Report usage back to backend for accurate quota tracking (fire-and-forget)
    fetch(`${BACKEND_URL}/api/v1/usage`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authCtx.entitlementToken}`,
      },
      body: JSON.stringify({
        entitlement_id: authCtx.entitlementId,
        agent_id: authCtx.agentId,
        buyer_id: authCtx.buyerId,
        input_tokens: runResult.input_tokens,
        output_tokens: runResult.output_tokens,
        cost_micros: runResult.cost_micros,
        status: "ok",
      }),
    }).catch((err) => {
      console.error("Failed to record usage:", err);
    });

    return {
      content: [
        {
          type: "text" as const,
          text: runResult.result,
        },
      ],
      isError: false,
    };
  };

  return { definition, handler };
}
