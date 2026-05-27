import { z } from "zod";
import { readFileSync, readdirSync, statSync } from "fs";
import { join, relative } from "path";
import { execSync } from "child_process";

export const fileOpsTool = {
  name: "agentx_file_ops",
  description: "Read files or list directory contents in the project",
  inputSchema: {
    operation: z.enum(["read", "list"]).describe("Operation: read a file or list directory"),
    path: z.string().optional().describe("File or directory path (relative to project root)"),
    pattern: z.string().optional().describe("Glob pattern for listing (default: all files)"),
  },
  handler: async (args: {
    operation: string;
    path?: string;
    pattern?: string;
  }) => {
    const cwd = process.cwd();
    const targetPath = args.path ? join(cwd, args.path) : cwd;

    try {
      if (args.operation === "read") {
        if (!args.path) {
          throw new Error("path is required for read operation");
        }
        const content = readFileSync(targetPath, "utf-8");
        return {
          content: [
            {
              type: "text" as const,
              text: content,
            },
          ],
        };
      } else if (args.operation === "list") {
        // Use git ls-files for respecting .gitignore
        try {
          const files = execSync(
            "git ls-files --cached --others --exclude-standard",
            { cwd: targetPath }
          ).toString().trim();

          return {
            content: [
              {
                type: "text" as const,
                text: files || "(empty directory)",
              },
            ],
          };
        } catch {
          // Fallback if not a git repo
          const entries = readdirSync(targetPath);
          return {
            content: [
              {
                type: "text" as const,
                text: entries.join("\n"),
              },
            ],
          };
        }
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
