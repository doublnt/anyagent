use anyhow::Result;
use clap::Subcommand;

#[derive(Subcommand)]
pub enum ConfigCommands {
    /// Get a config value
    Get {
        /// Config key
        key: String,
    },

    /// Set a config value
    Set {
        /// Config key
        key: String,
        /// Config value
        value: String,
    },
}

pub async fn run(cmd: ConfigCommands) -> Result<()> {
    match cmd {
        ConfigCommands::Get { key } => {
            match key.as_str() {
                "api_url" => {
                    let config = crate::config::GlobalConfig::load()?;
                    println!("{}", config.api_base_url());
                }
                "project.name" => {
                    let config = crate::config::project::ProjectConfig::load()?;
                    println!("{}", config.name);
                }
                "project.memory_enabled" => {
                    let config = crate::config::project::ProjectConfig::load()?;
                    println!("{}", config.settings.memory_enabled);
                }
                "project.trace_enabled" => {
                    let config = crate::config::project::ProjectConfig::load()?;
                    println!("{}", config.settings.trace_enabled);
                }
                "project.default_agent" => {
                    let config = crate::config::project::ProjectConfig::load()?;
                    match config.settings.default_agent {
                        Some(agent) => println!("{}", agent),
                        None => println!("(not set)"),
                    }
                }
                _ => {
                    anyhow::bail!("Unknown config key: {}", key);
                }
            }
        }
        ConfigCommands::Set { key, value } => {
            match key.as_str() {
                "api_url" => {
                    let mut config = crate::config::GlobalConfig::load()?;
                    config.api_url = Some(value.clone());
                    config.save()?;
                    println!("Set api_url = {}", value);
                }
                "project.name" => {
                    let mut config = crate::config::project::ProjectConfig::load()?;
                    config.name = value.clone();
                    config.save()?;
                    println!("Set project.name = {}", value);
                }
                "project.memory_enabled" => {
                    let mut config = crate::config::project::ProjectConfig::load()?;
                    let v: bool = value.parse()?;
                    config.settings.memory_enabled = v;
                    config.save()?;
                    println!("Set project.memory_enabled = {}", v);
                }
                "project.trace_enabled" => {
                    let mut config = crate::config::project::ProjectConfig::load()?;
                    let v: bool = value.parse()?;
                    config.settings.trace_enabled = v;
                    config.save()?;
                    println!("Set project.trace_enabled = {}", v);
                }
                "project.default_agent" => {
                    let mut config = crate::config::project::ProjectConfig::load()?;
                    config.settings.default_agent = Some(value.clone());
                    config.save()?;
                    println!("Set project.default_agent = {}", value);
                }
                _ => {
                    anyhow::bail!("Unknown config key: {}", key);
                }
            }
        }
    }

    Ok(())
}
