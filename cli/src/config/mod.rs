pub mod project;

use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

/// Global user config stored at ~/.agentx/config.yaml
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct GlobalConfig {
    pub api_url: Option<String>,
    pub token: Option<String>,
    pub user_id: Option<String>,
    pub email: Option<String>,
}

impl GlobalConfig {
    pub fn config_dir() -> Result<PathBuf> {
        let dir = dirs::home_dir()
            .ok_or_else(|| anyhow::anyhow!("Cannot determine home directory"))?
            .join(".agentx");
        Ok(dir)
    }

    pub fn config_path() -> Result<PathBuf> {
        Ok(Self::config_dir()?.join("config.yaml"))
    }

    pub fn load() -> Result<Self> {
        let path = Self::config_path()?;
        if !path.exists() {
            return Ok(Self::default());
        }
        let content = std::fs::read_to_string(&path)?;
        let config: Self = serde_yaml::from_str(&content)?;
        Ok(config)
    }

    pub fn save(&self) -> Result<()> {
        let dir = Self::config_dir()?;
        std::fs::create_dir_all(&dir)?;
        let content = serde_yaml::to_string(self)?;
        std::fs::write(Self::config_path()?, content)?;
        Ok(())
    }

    pub fn api_base_url(&self) -> &str {
        self.api_url.as_deref().unwrap_or("https://api.anyagent.dev")
    }

    pub fn is_authenticated(&self) -> bool {
        self.token.is_some()
    }
}
