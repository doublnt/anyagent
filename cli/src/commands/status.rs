use anyhow::Result;
use crate::config::{project::ProjectConfig, GlobalConfig};

pub async fn run() -> Result<()> {
    // Global status
    let global = GlobalConfig::load()?;
    println!("{} AnyAgent Status", console::style("●").cyan());
    println!();

    if global.is_authenticated() {
        println!("  Auth:    {}", console::style("Logged in").green());
    } else {
        println!("  Auth:    {}", console::style("Not logged in").yellow());
    }

    // Project status
    match ProjectConfig::load() {
        Ok(proj) => {
            println!("  Project: {}", proj.name);
            println!("  Agents:  {}", proj.agents.len());
            for agent in &proj.agents {
                println!("    - {}@{}", agent.name, agent.version);
            }
        }
        Err(_) => {
            println!("  Project: {}", console::style("Not initialized").yellow());
            println!("  Run {} to initialize", console::style("agentx init").cyan());
        }
    }

    Ok(())
}
