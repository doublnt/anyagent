import { z } from "zod";
import { readFileSync, writeFileSync, mkdirSync, existsSync, readdirSync } from "fs";
import { join } from "path";
import { randomUUID } from "crypto";

export const traceTool = {
  name: "agentx_trace",
  description: "Record trace spans for agent execution tracking",
  inputSchema: {
    operation: z.enum(["start", "span", "complete"]).describe("Trace operation"),
    trace_id: z.string().optional().describe("Trace ID (for span/complete)"),
    task: z.string().optional().describe("Task description (for start)"),
    tool_name: z.string().optional().describe("Tool name (for span)"),
    input_data: z.string().optional().describe("Tool input (for span)"),
    output_data: z.string().optional().describe("Tool output (for span)"),
    status: z.enum(["ok", "error"]).optional().describe("Span status (for span/complete)"),
  },
  handler: async (args: {
    operation: string;
    trace_id?: string;
    task?: string;
    tool_name?: string;
    input_data?: string;
    output_data?: string;
    status?: string;
  }) => {
    const cwd = process.cwd();
    const tracesDir = join(cwd, ".agentx", "traces");

    try {
      mkdirSync(tracesDir, { recursive: true });

      if (args.operation === "start") {
        if (!args.task) {
          throw new Error("task is required for start operation");
        }

        const traceId = randomUUID();
        const traceFile = join(tracesDir, `${traceId}.json`);

        const trace = {
          id: traceId,
          task: args.task,
          status: "running",
          started_at: new Date().toISOString(),
          spans: [],
        };

        writeFileSync(traceFile, JSON.stringify(trace, null, 2));

        return {
          content: [
            {
              type: "text" as const,
              text: JSON.stringify({ trace_id: traceId, status: "started" }),
            },
          ],
        };
      } else if (args.operation === "span") {
        if (!args.trace_id || !args.tool_name) {
          throw new Error("trace_id and tool_name are required for span operation");
        }

        const traceFile = join(tracesDir, `${args.trace_id}.json`);
        if (!existsSync(traceFile)) {
          throw new Error(`Trace not found: ${args.trace_id}`);
        }

        const trace = JSON.parse(readFileSync(traceFile, "utf-8"));
        trace.spans.push({
          span_id: randomUUID().slice(0, 8),
          tool_name: args.tool_name,
          input: args.input_data,
          output: args.output_data,
          status: args.status || "ok",
          timestamp: new Date().toISOString(),
        });

        writeFileSync(traceFile, JSON.stringify(trace, null, 2));

        return {
          content: [
            {
              type: "text" as const,
              text: `Span recorded for ${args.tool_name}`,
            },
          ],
        };
      } else if (args.operation === "complete") {
        if (!args.trace_id) {
          throw new Error("trace_id is required for complete operation");
        }

        const traceFile = join(tracesDir, `${args.trace_id}.json`);
        if (!existsSync(traceFile)) {
          throw new Error(`Trace not found: ${args.trace_id}`);
        }

        const trace = JSON.parse(readFileSync(traceFile, "utf-8"));
        trace.status = args.status || "completed";
        trace.finished_at = new Date().toISOString();

        writeFileSync(traceFile, JSON.stringify(trace, null, 2));

        return {
          content: [
            {
              type: "text" as const,
              text: `Trace ${args.trace_id} completed`,
            },
          ],
        };
      } else {
        throw new Error(`Unknown operation: ${args.operation}`);
      }
    } catch (err: any) {
      return {
        content: [
          {
            type: "text" as const,
            text: `Error: ${err.message}`,
          },
        ],
        isError: true,
      };
    }
  },
};
