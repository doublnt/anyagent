import { z } from "zod";
import { readFileSync, writeFileSync, mkdirSync, existsSync, readdirSync } from "fs";
import { join } from "path";
import { randomUUID } from "crypto";

export const memoryTool = {
  name: "agentx_memory",
  description: "Search or save project memories (facts, decisions, preferences, context)",
  inputSchema: {
    operation: z.enum(["search", "save", "list"]).describe("Operation: search, save, or list memories"),
    query: z.string().optional().describe("Search query (for search operation)"),
    kind: z.enum(["fact", "decision", "preference", "context"]).optional().describe("Memory kind (for save)"),
    content: z.string().optional().describe("Memory content (for save)"),
    limit: z.number().optional().describe("Max results for search (default: 5)"),
  },
  handler: async (args: {
    operation: string;
    query?: string;
    kind?: string;
    content?: string;
    limit?: number;
  }) => {
    const cwd = process.cwd();
    const memoryDir = join(cwd, ".agentx", "memory");

    try {
      if (args.operation === "save") {
        if (!args.kind || !args.content) {
          throw new Error("kind and content are required for save operation");
        }

        mkdirSync(memoryDir, { recursive: true });

        const id = randomUUID().slice(0, 8);
        const filename = `${id}.yaml`;
        const filepath = join(memoryDir, filename);

        const memory = {
          id: randomUUID(),
          kind: args.kind,
          content: args.content,
          source: "agent",
          created_at: new Date().toISOString(),
        };

        writeFileSync(filepath, JSON.stringify(memory, null, 2));

        return {
          content: [
            {
              type: "text" as const,
              text: `Memory saved: ${filename}`,
            },
          ],
        };
      } else if (args.operation === "search") {
        if (!args.query) {
          throw new Error("query is required for search operation");
        }

        if (!existsSync(memoryDir)) {
          return {
            content: [
              {
                type: "text" as const,
                text: "No memories found.",
              },
            ],
          };
        }

        // Simple keyword search (vector search requires cloud sync)
        const files = readdirSync(memoryDir).filter((f) => f.endsWith(".yaml"));
        const results: string[] = [];

        for (const file of files) {
          const content = readFileSync(join(memoryDir, file), "utf-8");
          if (content.toLowerCase().includes(args.query!.toLowerCase())) {
            results.push(content);
          }
        }

        return {
          content: [
            {
              type: "text" as const,
              text: results.length > 0
                ? results.slice(0, args.limit || 5).join("\n---\n")
                : "No matching memories found.",
            },
          ],
        };
      } else if (args.operation === "list") {
        if (!existsSync(memoryDir)) {
          return {
            content: [
              {
                type: "text" as const,
                text: "No memories yet.",
              },
            ],
          };
        }

        const files = readdirSync(memoryDir).filter((f) => f.endsWith(".yaml"));
        const memories = files.map((f) => {
          const content = readFileSync(join(memoryDir, f), "utf-8");
          return `--- ${f} ---\n${content}`;
        });

        return {
          content: [
            {
              type: "text" as const,
              text: memories.length > 0 ? memories.join("\n\n") : "No memories yet.",
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
