import { z } from "zod";
import { execSync } from "child_process";

export const runCommandTool = {
  name: "agentx_run_command",
  description: "Run a shell command in the project directory (restricted to safe commands)",
  inputSchema: {
    command: z.string().describe("Command to execute"),
    timeout_ms: z.number().optional().describe("Timeout in milliseconds (default: 30000)"),
  },
  handler: async (args: {
    command: string;
    timeout_ms?: number;
  }) => {
    const cwd = process.cwd();
    const timeout = args.timeout_ms || 30000;

    // Safety: block dangerous commands
    const blocked = ["rm -rf", "mkfs", "dd if=", ":(){", "fork bomb"];
    const cmdLower = args.command.toLowerCase();
    for (const pattern of blocked) {
      if (cmdLower.includes(pattern)) {
        return {
          content: [
            {
              type: "text" as const,
              text: `Error: Command blocked for safety: ${args.command}`,
            },
          ],
          isError: true,
        };
      }
    }

    try {
      const output = execSync(args.command, {
        cwd,
        timeout,
        encoding: "utf-8",
        maxBuffer: 1024 * 1024, // 1MB
      });

      return {
        content: [
          {
            type: "text" as const,
            text: output || "(no output)",
          },
        ],
      };
    } catch (err: any) {
      return {
        content: [
          {
            type: "text" as const,
            text: `Exit code: ${err.status}\nStdout: ${err.stdout || ""}\nStderr: ${err.stderr || ""}`,
          },
        ],
        isError: true,
      };
    }
  },
};
