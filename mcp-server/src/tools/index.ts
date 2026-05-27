import { gitContextTool } from "./git-context.js";
import { fileOpsTool } from "./file-ops.js";
import { memoryTool } from "./memory.js";
import { traceTool } from "./trace.js";
import { runCommandTool } from "./run-command.js";

export const tools = [
  gitContextTool,
  fileOpsTool,
  memoryTool,
  traceTool,
  runCommandTool,
];
