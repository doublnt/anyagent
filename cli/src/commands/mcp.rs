use anyhow::Result;
use clap::Args;
use std::io::{self, BufRead, Write};

#[derive(Args)]
pub struct McpArgs {
    /// Transport mode
    #[arg(long, default_value = "stdio")]
    transport: String,

    /// Port for HTTP transport
    #[arg(long, default_value = "3000")]
    port: u16,
}

pub async fn run(args: McpArgs) -> Result<()> {
    match args.transport.as_str() {
        "stdio" => {
            eprintln!("{} Starting MCP server (stdio)...", console::style("●").cyan());
            eprintln!("Connect with: claude mcp add agentx -- agentx mcp");
            eprintln!();
            run_stdio_server().await
        }
        "http" => {
            eprintln!("{} Starting MCP server (HTTP on port {})...", console::style("●").cyan(), args.port);
            eprintln!("{} HTTP transport not yet implemented.", console::style("TODO").yellow());
            Ok(())
        }
        _ => {
            anyhow::bail!("Unknown transport: {}. Use 'stdio' or 'http'.", args.transport);
        }
    }
}

async fn run_stdio_server() -> Result<()> {
    let stdin = io::stdin();
    let stdout = io::stdout();
    let mut reader = io::BufReader::new(stdin.lock());
    let mut writer = io::BufWriter::new(stdout.lock());

    // Send initialize response
    let init_response = serde_json::json!({
        "jsonrpc": "2.0",
        "id": 0,
        "result": {
            "protocolVersion": "2024-11-05",
            "capabilities": {
                "tools": {}
            },
            "serverInfo": {
                "name": "anyagent",
                "version": "0.1.0"
            }
        }
    });

    let mut line = String::new();
    loop {
        line.clear();
        match reader.read_line(&mut line) {
            Ok(0) => break, // EOF
            Ok(_) => {}
            Err(e) => {
                eprintln!("Error reading: {}", e);
                break;
            }
        }

        let msg: serde_json::Value = match serde_json::from_str(line.trim()) {
            Ok(v) => v,
            Err(e) => {
                eprintln!("Invalid JSON: {}", e);
                continue;
            }
        };

        let method = msg.get("method").and_then(|m| m.as_str()).unwrap_or("");
        let id = msg.get("id").cloned();

        let response = match method {
            "initialize" => {
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": id,
                    "result": {
                        "protocolVersion": "2024-11-05",
                        "capabilities": {
                            "tools": {}
                        },
                        "serverInfo": {
                            "name": "anyagent",
                            "version": "0.1.0"
                        }
                    }
                })
            }
            "tools/list" => {
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": id,
                    "result": {
                        "tools": get_tools()
                    }
                })
            }
            "tools/call" => {
                let params = msg.get("params").cloned().unwrap_or_default();
                let tool_name = params.get("name").and_then(|n| n.as_str()).unwrap_or("");
                let tool_args = params.get("arguments").cloned().unwrap_or_default();

                match handle_tool_call(tool_name, tool_args).await {
                    Ok(result) => {
                        serde_json::json!({
                            "jsonrpc": "2.0",
                            "id": id,
                            "result": result
                        })
                    }
                    Err(e) => {
                        serde_json::json!({
                            "jsonrpc": "2.0",
                            "id": id,
                            "error": {
                                "code": -32000,
                                "message": e.to_string()
                            }
                        })
                    }
                }
            }
            "notifications/initialized" => {
                // No response needed for notifications
                continue;
            }
            _ => {
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": id,
                    "error": {
                        "code": -32601,
                        "message": format!("Method not found: {}", method)
                    }
                })
            }
        };

        let response_str = serde_json::to_string(&response)?;
        writer.write_all(response_str.as_bytes())?;
        writer.write_all(b"\n")?;
        writer.flush()?;
    }

    Ok(())
}

fn get_tools() -> Vec<serde_json::Value> {
    vec![
        serde_json::json!({
            "name": "agentx_git_context",
            "description": "Get git context: branch, status, diff, recent commits",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "include_diff": { "type": "boolean", "description": "Include git diff", "default": true },
                    "diff_mode": { "type": "string", "enum": ["staged", "unstaged", "all"], "default": "all" }
                }
            }
        }),
        serde_json::json!({
            "name": "agentx_read_file",
            "description": "Read a file from the project",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "path": { "type": "string", "description": "File path relative to project root" }
                },
                "required": ["path"]
            }
        }),
        serde_json::json!({
            "name": "agentx_list_files",
            "description": "List files in the project",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "pattern": { "type": "string", "description": "Glob pattern (default: all files)" }
                }
            }
        }),
        serde_json::json!({
            "name": "agentx_memory",
            "description": "Search or save project memories",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "operation": { "type": "string", "enum": ["search", "save", "list"] },
                    "query": { "type": "string", "description": "Search query" },
                    "kind": { "type": "string", "enum": ["fact", "decision", "preference", "context"] },
                    "content": { "type": "string", "description": "Memory content to save" }
                },
                "required": ["operation"]
            }
        }),
        serde_json::json!({
            "name": "agentx_run_command",
            "description": "Run a shell command in the project directory",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "command": { "type": "string", "description": "Command to execute" },
                    "timeout_ms": { "type": "integer", "description": "Timeout in ms", "default": 30000 }
                },
                "required": ["command"]
            }
        }),
    ]
}

async fn handle_tool_call(name: &str, args: serde_json::Value) -> Result<serde_json::Value> {
    match name {
        "agentx_git_context" => {
            let cwd = std::env::current_dir()?;
            let ctx = crate::runner::git::GitContext::collect(&cwd)?;

            let include_diff = args.get("include_diff")
                .and_then(|v| v.as_bool())
                .unwrap_or(true);

            let mut result = serde_json::json!({
                "branch": ctx.branch,
                "remote": ctx.remote_url,
                "changed_files": ctx.status.len(),
            });

            if let Some(commit) = &ctx.last_commit {
                result["last_commit"] = serde_json::json!({
                    "hash": commit.hash,
                    "message": commit.message,
                });
            }

            if include_diff {
                let diff_mode = args.get("diff_mode")
                    .and_then(|v| v.as_str())
                    .unwrap_or("all");

                let diff_cmd = match diff_mode {
                    "staged" => "git diff --cached",
                    "unstaged" => "git diff",
                    _ => "git diff HEAD",
                };

                if let Ok(output) = std::process::Command::new("sh")
                    .args(["-c", diff_cmd])
                    .current_dir(&cwd)
                    .output()
                {
                    let diff = String::from_utf8_lossy(&output.stdout);
                    let max_lines = 500;
                    let lines: Vec<&str> = diff.lines().take(max_lines).collect();
                    result["diff"] = serde_json::Value::String(lines.join("\n"));
                }
            }

            Ok(serde_json::json!({
                "content": [{
                    "type": "text",
                    "text": serde_json::to_string_pretty(&result)?
                }]
            }))
        }
        "agentx_read_file" => {
            let path = args.get("path")
                .and_then(|v| v.as_str())
                .ok_or_else(|| anyhow::anyhow!("path is required"))?;

            let cwd = std::env::current_dir()?;
            let full_path = cwd.join(path);

            let content = std::fs::read_to_string(&full_path)?;
            Ok(serde_json::json!({
                "content": [{
                    "type": "text",
                    "text": content
                }]
            }))
        }
        "agentx_list_files" => {
            let cwd = std::env::current_dir()?;
            let files = crate::runner::fs::list_files(&cwd, "**/*")?;

            Ok(serde_json::json!({
                "content": [{
                    "type": "text",
                    "text": files.join("\n")
                }]
            }))
        }
        "agentx_memory" => {
            let operation = args.get("operation")
                .and_then(|v| v.as_str())
                .ok_or_else(|| anyhow::anyhow!("operation is required"))?;

            let cwd = std::env::current_dir()?;
            let memory_dir = cwd.join(".agentx").join("memory");

            match operation {
                "save" => {
                    let kind = args.get("kind")
                        .and_then(|v| v.as_str())
                        .unwrap_or("fact");
                    let content = args.get("content")
                        .and_then(|v| v.as_str())
                        .ok_or_else(|| anyhow::anyhow!("content is required"))?;

                    std::fs::create_dir_all(&memory_dir)?;
                    let id = uuid::Uuid::new_v4().to_string();
                    let filename = format!("{}.yaml", &id[..8]);

                    let memory = serde_json::json!({
                        "id": id,
                        "kind": kind,
                        "content": content,
                        "source": "agent",
                        "created_at": chrono::Utc::now().to_rfc3339(),
                    });

                    std::fs::write(
                        memory_dir.join(&filename),
                        serde_json::to_string_pretty(&memory)?,
                    )?;

                    Ok(serde_json::json!({
                        "content": [{
                            "type": "text",
                            "text": format!("Memory saved: {}", filename)
                        }]
                    }))
                }
                "search" => {
                    let query = args.get("query")
                        .and_then(|v| v.as_str())
                        .ok_or_else(|| anyhow::anyhow!("query is required"))?;

                    if !memory_dir.exists() {
                        return Ok(serde_json::json!({
                            "content": [{
                                "type": "text",
                                "text": "No memories found."
                            }]
                        }));
                    }

                    let mut results = Vec::new();
                    for entry in std::fs::read_dir(&memory_dir)? {
                        let entry = entry?;
                        if entry.path().extension().map_or(false, |e| e == "yaml") {
                            let content = std::fs::read_to_string(entry.path())?;
                            if content.to_lowercase().contains(&query.to_lowercase()) {
                                results.push(content);
                            }
                        }
                    }

                    Ok(serde_json::json!({
                        "content": [{
                            "type": "text",
                            "text": if results.is_empty() {
                                "No matching memories found.".to_string()
                            } else {
                                results.join("\n---\n")
                            }
                        }]
                    }))
                }
                "list" => {
                    if !memory_dir.exists() {
                        return Ok(serde_json::json!({
                            "content": [{
                                "type": "text",
                                "text": "No memories yet."
                            }]
                        }));
                    }

                    let mut memories = Vec::new();
                    for entry in std::fs::read_dir(&memory_dir)? {
                        let entry = entry?;
                        if entry.path().extension().map_or(false, |e| e == "yaml") {
                            let content = std::fs::read_to_string(entry.path())?;
                            memories.push(content);
                        }
                    }

                    Ok(serde_json::json!({
                        "content": [{
                            "type": "text",
                            "text": if memories.is_empty() {
                                "No memories yet.".to_string()
                            } else {
                                memories.join("\n---\n")
                            }
                        }]
                    }))
                }
                _ => anyhow::bail!("Unknown operation: {}", operation),
            }
        }
        "agentx_run_command" => {
            let command = args.get("command")
                .and_then(|v| v.as_str())
                .ok_or_else(|| anyhow::anyhow!("command is required"))?;

            let cwd = std::env::current_dir()?;
            let output = std::process::Command::new("sh")
                .args(["-c", command])
                .current_dir(&cwd)
                .output()?;

            let stdout = String::from_utf8_lossy(&output.stdout);
            let stderr = String::from_utf8_lossy(&output.stderr);

            let result = if output.status.success() {
                stdout.to_string()
            } else {
                format!("Exit code: {}\nStdout: {}\nStderr: {}",
                    output.status.code().unwrap_or(-1), stdout, stderr)
            };

            Ok(serde_json::json!({
                "content": [{
                    "type": "text",
                    "text": result
                }],
                "isError": !output.status.success()
            }))
        }
        _ => anyhow::bail!("Unknown tool: {}", name),
    }
}
