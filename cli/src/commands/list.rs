use anyhow::Result;
use crate::config::project::ProjectConfig;

pub async fn run() -> Result<()> {
    let config = ProjectConfig::load()?;

    if config.agents.is_empty() {
        println!("No agents installed.");
        println!("Run {} to install an agent.", console::style("agentx install <name>").cyan());
        return Ok(());
    }

    println!("Installed agents:");
    println!();
    for agent in &config.agents {
        println!(
            "  {} {}",
            console::style(&agent.name).green(),
            console::style(format!("@{}", agent.version)).dim()
        );
    }

    Ok(())
}
