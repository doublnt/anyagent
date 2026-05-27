use anyhow::Result;
use clap::Args;
use crate::config::project::{ProjectConfig, ProjectSettings};

#[derive(Args)]
pub struct InitArgs {
    /// Project name (defaults to directory name)
    #[arg(long)]
    name: Option<String>,

    /// Template agent pack to install
    #[arg(long)]
    template: Option<String>,

    /// Force reinitialize if .agentx/ already exists
    #[arg(long)]
    force: bool,
}

pub async fn run(args: InitArgs) -> Result<()> {
    let cwd = std::env::current_dir()?;
    let agentx_dir = cwd.join(".agentx");

    if agentx_dir.exists() && !args.force {
        anyhow::bail!(
            ".agentx/ already exists. Use --force to reinitialize."
        );
    }

    // Create directory structure
    std::fs::create_dir_all(&agentx_dir)?;
    std::fs::create_dir_all(agentx_dir.join("agents"))?;
    std::fs::create_dir_all(agentx_dir.join("memory"))?;
    std::fs::create_dir_all(agentx_dir.join("traces"))?;

    // Determine project name
    let name = args.name.unwrap_or_else(|| {
        cwd.file_name()
            .map(|n| n.to_string_lossy().to_string())
            .unwrap_or_else(|| "my-project".to_string())
    });

    // Create config
    let config = ProjectConfig {
        project_id: None,
        name: name.clone(),
        agents: vec![],
        settings: ProjectSettings::default(),
    };
    config.save()?;

    // Add .agentx/ to .gitignore if not already there
    let gitignore = cwd.join(".gitignore");
    let entry = ".agentx/";
    if gitignore.exists() {
        let content = std::fs::read_to_string(&gitignore)?;
        if !content.contains(entry) {
            let mut content = content;
            if !content.ends_with('\n') {
                content.push('\n');
            }
            content.push_str(entry);
            content.push('\n');
            std::fs::write(&gitignore, content)?;
        }
    } else {
        std::fs::write(&gitignore, format!("{}\n", entry))?;
    }

    println!("{} Initialized agentx project: {}", console::style("✓").green(), name);
    println!();
    println!("  {}", console::style(".agentx/").dim());
    println!("  ├── config.yaml");
    println!("  ├── agents/");
    println!("  ├── memory/");
    println!("  └── traces/");
    println!();
    println!("Next steps:");
    println!("  1. {} to authenticate", console::style("agentx login").cyan());
    println!("  2. {} to install an agent", console::style("agentx install <agent>").cyan());
    println!("  3. {} to start the MCP server", console::style("agentx mcp").cyan());

    Ok(())
}
