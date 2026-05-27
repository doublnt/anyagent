use anyhow::Result;
use clap::Args;
use crate::config::project::ProjectConfig;

#[derive(Args)]
pub struct UninstallArgs {
    /// Agent name to uninstall
    agent: String,
}

pub async fn run(args: UninstallArgs) -> Result<()> {
    let mut config = ProjectConfig::load()?;

    let idx = config.agents.iter().position(|a| a.name == args.agent);
    match idx {
        Some(i) => {
            config.agents.remove(i);
            config.save()?;

            // Remove agent directory
            let agents_dir = ProjectConfig::agents_dir()?.join(&args.agent);
            if agents_dir.exists() {
                std::fs::remove_dir_all(&agents_dir)?;
            }

            println!("{} Uninstalled {}", console::style("✓").green(), args.agent);
        }
        None => {
            println!("Agent {} is not installed.", args.agent);
        }
    }

    Ok(())
}
