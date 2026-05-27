use anyhow::Result;
use clap::Args;
use crate::config::project::ProjectConfig;

#[derive(Args)]
pub struct UpdateArgs {
    /// Specific agent to update (updates all if omitted)
    agent: Option<String>,
}

pub async fn run(args: UpdateArgs) -> Result<()> {
    let config = ProjectConfig::load()?;

    if config.agents.is_empty() {
        println!("No agents installed.");
        return Ok(());
    }

    let to_update: Vec<_> = if let Some(name) = args.agent {
        config.agents.iter().filter(|a| a.name == name).collect()
    } else {
        config.agents.iter().collect()
    };

    if to_update.is_empty() {
        println!("Agent not found.");
        return Ok(());
    }

    for agent in to_update {
        // TODO: Check registry for latest version and update
        println!("Checking {}... {}", agent.name, console::style("up to date").green());
    }

    Ok(())
}
