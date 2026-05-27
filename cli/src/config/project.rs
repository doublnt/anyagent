use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

/// Project config stored at .agentx/config.yaml
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProjectConfig {
    pub project_id: Option<String>,
    pub name: String,
    #[serde(default)]
    pub agents: Vec<InstalledAgent>,
    #[serde(default)]
    pub settings: ProjectSettings,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InstalledAgent {
    pub name: String,
    pub version: String,
    pub installed_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ProjectSettings {
    #[serde(default = "default_memory_enabled")]
    pub memory_enabled: bool,
    #[serde(default = "default_trace_enabled")]
    pub trace_enabled: bool,
    pub default_agent: Option<String>,
}

fn default_memory_enabled() -> bool {
    true
}

fn default_trace_enabled() -> bool {
    true
}

impl ProjectConfig {
    pub fn project_dir() -> Result<PathBuf> {
        Ok(std::env::current_dir()?.join(".agentx"))
    }

    pub fn config_path() -> Result<PathBuf> {
        Ok(Self::project_dir()?.join("config.yaml"))
    }

    pub fn find_project_root() -> Result<PathBuf> {
        let mut dir = std::env::current_dir()?;
        loop {
            if dir.join(".agentx").join("config.yaml").exists() {
                return Ok(dir);
            }
            if !dir.pop() {
                anyhow::bail!("Not in an agentx project. Run `agentx init` first.");
            }
        }
    }

    pub fn load() -> Result<Self> {
        let path = Self::config_path()?;
        if !path.exists() {
            anyhow::bail!("Not in an agentx project. Run `agentx init` first.");
        }
        let content = std::fs::read_to_string(&path)?;
        let config: Self = serde_yaml::from_str(&content)?;
        Ok(config)
    }

    pub fn save(&self) -> Result<()> {
        let dir = Self::project_dir()?;
        std::fs::create_dir_all(&dir)?;
        let content = serde_yaml::to_string(self)?;
        std::fs::write(Self::config_path()?, content)?;
        Ok(())
    }

    pub fn agents_dir() -> Result<PathBuf> {
        Ok(Self::project_dir()?.join("agents"))
    }

    pub fn memory_dir() -> Result<PathBuf> {
        Ok(Self::project_dir()?.join("memory"))
    }

    pub fn traces_dir() -> Result<PathBuf> {
        Ok(Self::project_dir()?.join("traces"))
    }

    pub fn has_agent(&self, name: &str) -> bool {
        self.agents.iter().any(|a| a.name == name)
    }
}
