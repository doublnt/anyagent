import { z } from "zod";
import { execSync } from "child_process";

export const gitContextTool = {
  name: "agentx_git_context",
  description: "Get git context for the current project: branch, status, recent commits, diff",
  inputSchema: {
    include_diff: z.boolean().optional().describe("Include git diff (default: true)"),
    diff_mode: z.enum(["staged", "unstaged", "all"]).optional().describe("Diff mode (default: all)"),
    max_diff_lines: z.number().optional().describe("Max diff lines (default: 500)"),
  },
  handler: async (args: {
    include_diff?: boolean;
    diff_mode?: string;
    max_diff_lines?: number;
  }) => {
    const cwd = process.cwd();
    const includeDiff = args.include_diff !== false;
    const diffMode = args.diff_mode || "all";
    const maxLines = args.max_diff_lines || 500;

    try {
      // Get branch
      const branch = execSync("git rev-parse --abbrev-ref HEAD", { cwd })
        .toString().trim();

      // Get status
      const status = execSync("git status --porcelain", { cwd })
        .toString().trim();

      // Get last 5 commits
      const commits = execSync(
        'git log -5 --format="%h %s (%an, %ar)"',
        { cwd }
      ).toString().trim();

      let diff = "";
      if (includeDiff) {
        const diffCmd =
          diffMode === "staged" ? "git diff --cached" :
          diffMode === "unstaged" ? "git diff" :
          "git diff HEAD";

        diff = execSync(`${diffCmd} | head -n ${maxLines}`, { cwd })
          .toString().trim();

        if (diff.length === 0) {
          diff = "(no changes)";
        }
      }

      return {
        content: [
          {
            type: "text" as const,
            text: JSON.stringify({ branch, status, commits, diff }, null, 2),
          },
        ],
      };
    } catch (err: any) {
      return {
        content: [
          {
            type: "text" as const,
            text: `Error getting git context: ${err.message}`,
          },
        ],
        isError: true,
      };
    }
  },
};
