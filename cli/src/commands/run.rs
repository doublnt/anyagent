use anyhow::Result;
use clap::Args;
use crate::config::project::ProjectConfig;
use crate::runner::{git, fs, analyzer};

#[derive(Args)]
pub struct RunArgs {
    /// Task description
    task: Vec<String>,

    /// Specific agent to use
    #[arg(long)]
    agent: Option<String>,

    /// Dry run - show execution plan without running
    #[arg(long)]
    dry_run: bool,

    /// Output format: text, json, context
    #[arg(long, default_value = "text")]
    format: String,
}

pub async fn run(args: RunArgs) -> Result<()> {
    let config = ProjectConfig::load()?;
    let task = args.task.join(" ");

    if task.is_empty() {
        anyhow::bail!("Please provide a task description.");
    }

    // Determine which agent to use
    let agent_name = args.agent
        .or(config.settings.default_agent)
        .or_else(|| config.agents.first().map(|a| a.name.clone()));
    let agent_name = match agent_name {
        Some(name) => name,
        None => anyhow::bail!("No agents installed. Run: agentx install <agent-name>"),
    };
    let agent_name = agent_name.as_str();

    // Check if agent is installed
    let agent_dir = ProjectConfig::agents_dir()?.join(agent_name);
    if !agent_dir.exists() {
        anyhow::bail!(
            "Agent '{}' not installed. Run: agentx install {}",
            agent_name, agent_name
        );
    }

    // Load agent manifest
    let manifest_path = agent_dir.join("agent.yaml");
    let manifest = if manifest_path.exists() {
        let content = std::fs::read_to_string(&manifest_path)?;
        parse_manifest(&content)
    } else {
        ManifestInfo {
            name: agent_name.to_string(),
            version: "unknown".to_string(),
            description: None,
            prompts: vec![],
        }
    };

    if args.dry_run {
        println!("{} Dry run", console::style("●").cyan());
        println!();
        println!("  Task:    {}", task);
        println!("  Agent:   {}@{}", manifest.name, manifest.version);
        if let Some(desc) = &manifest.description {
            println!("  Desc:    {}", desc);
        }
        println!();
        println!("This would:");
        println!("  1. Load agent pack: {}", manifest.name);
        println!("  2. Collect project context (git, files)");
        println!("  3. Build prompt with context");
        println!("  4. Execute with local runner or MCP");
        println!("  5. Record trace");
        return Ok(());
    }

    // Collect project context
    println!("{} Collecting project context...", console::style("●").cyan());

    let cwd = std::env::current_dir()?;
    let git_context = git::GitContext::collect(&cwd)?;

    // Print git context
    println!();
    println!("{} Git Context", console::style("──").dim());
    if let Some(branch) = &git_context.branch {
        println!("  Branch: {}", branch);
    }
    if let Some(remote) = &git_context.remote_url {
        println!("  Remote: {}", remote);
    }
    if let Some(commit) = &git_context.last_commit {
        println!("  Last commit: {} - {}", &commit.hash[..7], commit.message);
    }

    if !git_context.status.is_empty() {
        println!("  Changed files:");
        for status in git_context.status.iter().take(10) {
            println!("    {} {}", status.status, status.path);
        }
        if git_context.status.len() > 10 {
            println!("    ... and {} more", git_context.status.len() - 10);
        }
    }

    // Get file tree
    println!();
    println!("{} Project Structure", console::style("──").dim());
    let tree = analyzer::get_tree(&cwd, 2)?;
    for line in tree.lines().take(20) {
        println!("  {}", line);
    }

    // Load agent prompts
    if !manifest.prompts.is_empty() {
        println!();
        println!("{} Agent Prompts", console::style("──").dim());
        for prompt in &manifest.prompts {
            let prompt_path = agent_dir.join(&prompt.path);
            if prompt_path.exists() {
                let content = std::fs::read_to_string(&prompt_path)?;
                println!("  [{}]", prompt.name);
                // Show first few lines of prompt
                for line in content.lines().take(5) {
                    println!("    {}", line);
                }
                if content.lines().count() > 5 {
                    println!("    ... ({} lines total)", content.lines().count());
                }
            }
        }
    }

    // Output as context for MCP
    println!();
    println!("{} Ready", console::style("──").dim());
    println!();
    println!("To execute this task with Claude Code:");
    println!("  1. Start MCP server: {}", console::style("agentx mcp").cyan());
    println!("  2. Use Claude Code with the agent context");
    println!();
    println!("Or use the MCP tools directly in Claude Code to:");
    println!("  - Search memories: agentx_memory search \"{}\"", task);
    println!("  - Record trace: agentx_trace start --task \"{}\"", task);

    // Save context to .agentx/traces/ for later use
    let trace_dir = ProjectConfig::traces_dir()?;
    std::fs::create_dir_all(&trace_dir)?;

    let trace_id = uuid::Uuid::new_v4().to_string();
    let trace_file = trace_dir.join(format!("{}.json", &trace_id[..8]));

    let trace = serde_json::json!({
        "id": trace_id,
        "task": task,
        "agent": agent_name,
        "status": "prepared",
        "created_at": chrono::Utc::now().to_rfc3339(),
        "context": {
            "branch": git_context.branch,
            "remote": git_context.remote_url,
            "changed_files": git_context.status.len(),
        }
    });

    std::fs::write(&trace_file, serde_json::to_string_pretty(&trace)?)?;
    println!();
    println!("{} Trace saved: {}", console::style("✓").green(), trace_file.display());

    Ok(())
}

struct ManifestInfo {
    name: String,
    version: String,
    description: Option<String>,
    prompts: Vec<PromptInfo>,
}

struct PromptInfo {
    name: String,
    path: String,
}

fn parse_manifest(content: &str) -> ManifestInfo {
    let mut name = String::new();
    let mut version = String::new();
    let mut description = None;
    let mut prompts = Vec::new();

    for line in content.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }

        if let Some(rest) = line.strip_prefix("name:") {
            name = rest.trim().to_string();
        } else if let Some(rest) = line.strip_prefix("version:") {
            version = rest.trim().to_string();
        } else if let Some(rest) = line.strip_prefix("description:") {
            description = Some(rest.trim().trim_matches('"').to_string());
        }
    }

    ManifestInfo {
        name,
        version,
        description,
        prompts,
    }
}
