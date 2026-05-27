use anyhow::{Context, Result};
use clap::Subcommand;
use crate::config::{project::ProjectConfig, GlobalConfig};

#[derive(Subcommand)]
pub enum MemoryCommands {
    /// List project memories
    List {
        /// List from cloud instead of local
        #[arg(long)]
        cloud: bool,
    },

    /// Add a new memory
    Add {
        /// Memory content
        content: String,

        /// Memory kind: fact, decision, preference, context
        #[arg(long, default_value = "fact")]
        kind: String,
    },

    /// Search memories
    Search {
        /// Search query
        query: String,

        /// Max results
        #[arg(long, default_value = "5")]
        limit: u32,

        /// Search in cloud
        #[arg(long)]
        cloud: bool,
    },

    /// Sync memories to cloud
    Sync,

    /// Import memories from file
    Import {
        /// File path
        file: String,
    },
}

pub async fn run(cmd: MemoryCommands) -> Result<()> {
    let config = ProjectConfig::load()?;

    match cmd {
        MemoryCommands::List { cloud } => {
            if cloud {
                list_cloud_memories().await?;
            } else {
                list_local_memories()?;
            }
        }
        MemoryCommands::Add { content, kind } => {
            add_memory(&content, &kind).await?;
        }
        MemoryCommands::Search { query, limit, cloud } => {
            if cloud {
                search_cloud_memories(&query, limit).await?;
            } else {
                search_local_memories(&query)?;
            }
        }
        MemoryCommands::Sync => {
            sync_memories().await?;
        }
        MemoryCommands::Import { file } => {
            import_memory(&file).await?;
        }
    }

    Ok(())
}

fn list_local_memories() -> Result<()> {
    let memory_dir = ProjectConfig::memory_dir()?;
    if !memory_dir.exists() {
        println!("No memories yet.");
        return Ok(());
    }

    let mut entries: Vec<_> = std::fs::read_dir(&memory_dir)?
        .filter_map(|e| e.ok())
        .filter(|e| e.path().extension().map_or(false, |ext| ext == "yaml"))
        .collect();

    entries.sort_by_key(|e| e.file_name());

    if entries.is_empty() {
        println!("No memories yet.");
        println!("Run {} to add one.", console::style("agentx memory add \"...\"").cyan());
        return Ok(());
    }

    println!("Local memories:");
    println!();
    for entry in entries {
        let content = std::fs::read_to_string(entry.path())?;
        let name = entry.file_name().to_string_lossy().to_string();
        // Parse kind and content from YAML-like format
        let kind = extract_field(&content, "kind").unwrap_or_else(|| "unknown".to_string());
        let text = extract_field(&content, "content").unwrap_or_else(|| content.clone());
        println!("  {} [{}] {}", console::style(&name).green(), console::style(&kind).yellow(), text);
    }

    Ok(())
}

async fn list_cloud_memories() -> Result<()> {
    let config = GlobalConfig::load()?;
    if !config.is_authenticated() {
        anyhow::bail!("Not logged in. Run `agentx login` first.");
    }

    let project_config = ProjectConfig::load()?;
    let project_id = project_config.project_id.as_deref().unwrap_or("default");

    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/projects/{}/memories", config.api_base_url(), project_id);

    let resp = client
        .get(&url)
        .bearer_auth(config.token.as_ref().unwrap())
        .send()
        .await
        .context("Failed to connect to API")?;

    if !resp.status().is_success() {
        anyhow::bail!("API error: {}", resp.status());
    }

    let memories: Vec<serde_json::Value> = resp.json().await?;

    if memories.is_empty() {
        println!("No cloud memories found.");
        return Ok(());
    }

    println!("Cloud memories:");
    println!();
    for m in &memories {
        let kind = m["kind"].as_str().unwrap_or("unknown");
        let content = m["content"].as_str().unwrap_or("");
        println!("  [{}] {}", console::style(kind).yellow(), content);
    }

    Ok(())
}

async fn add_memory(content: &str, kind: &str) -> Result<()> {
    let memory_dir = ProjectConfig::memory_dir()?;
    std::fs::create_dir_all(&memory_dir)?;

    let id = uuid::Uuid::new_v4().to_string();
    let filename = format!("{}.yaml", &id[..8]);
    let filepath = memory_dir.join(&filename);

    let memory = serde_yaml::to_string(&serde_json::json!({
        "id": id,
        "kind": kind,
        "content": content,
        "source": "user",
        "created_at": chrono::Utc::now().to_rfc3339(),
    }))?;

    std::fs::write(&filepath, memory)?;
    println!("{} Memory saved locally: {}", console::style("✓").green(), filename);
    println!("Run {} to sync to cloud.", console::style("agentx memory sync").cyan());

    Ok(())
}

fn search_local_memories(query: &str) -> Result<()> {
    let memory_dir = ProjectConfig::memory_dir()?;
    if !memory_dir.exists() {
        println!("No memories found.");
        return Ok(());
    }

    let mut found = 0;
    for entry in std::fs::read_dir(&memory_dir)? {
        let entry = entry?;
        if !entry.path().extension().map_or(false, |e| e == "yaml") {
            continue;
        }

        let content = std::fs::read_to_string(entry.path())?;
        if content.to_lowercase().contains(&query.to_lowercase()) {
            let name = entry.file_name().to_string_lossy().to_string();
            let text = extract_field(&content, "content").unwrap_or_else(|| content.clone());
            println!("  {} {}", console::style(&name).green(), text);
            found += 1;
        }
    }

    if found == 0 {
        println!("No matching memories found locally.");
        println!("Try {} for cloud search.", console::style("agentx memory search \"...\" --cloud").cyan());
    }

    Ok(())
}

async fn search_cloud_memories(query: &str, limit: u32) -> Result<()> {
    let config = GlobalConfig::load()?;
    if !config.is_authenticated() {
        anyhow::bail!("Not logged in. Run `agentx login` first.");
    }

    let project_config = ProjectConfig::load()?;
    let project_id = project_config.project_id.as_deref().unwrap_or("default");

    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/projects/{}/memories/search", config.api_base_url(), project_id);

    let resp = client
        .post(&url)
        .bearer_auth(config.token.as_ref().unwrap())
        .json(&serde_json::json!({
            "query": query,
            "limit": limit,
        }))
        .send()
        .await
        .context("Failed to connect to API")?;

    if !resp.status().is_success() {
        anyhow::bail!("API error: {}", resp.status());
    }

    let memories: Vec<serde_json::Value> = resp.json().await?;

    if memories.is_empty() {
        println!("No matching memories found.");
        return Ok(());
    }

    println!("Search results:");
    println!();
    for m in &memories {
        let kind = m["kind"].as_str().unwrap_or("unknown");
        let content = m["content"].as_str().unwrap_or("");
        println!("  [{}] {}", console::style(kind).yellow(), content);
    }

    Ok(())
}

async fn sync_memories() -> Result<()> {
    let config = GlobalConfig::load()?;
    if !config.is_authenticated() {
        anyhow::bail!("Not logged in. Run `agentx login` first.");
    }

    let memory_dir = ProjectConfig::memory_dir()?;
    if !memory_dir.exists() {
        println!("No local memories to sync.");
        return Ok(());
    }

    let project_config = ProjectConfig::load()?;
    let project_id = project_config.project_id.as_deref().unwrap_or("default");

    let client = reqwest::Client::new();
    let base_url = config.api_base_url().to_string();
    let token = config.token.as_ref().unwrap().clone();

    let mut synced = 0;
    let mut skipped = 0;

    for entry in std::fs::read_dir(&memory_dir)? {
        let entry = entry?;
        if !entry.path().extension().map_or(false, |e| e == "yaml") {
            continue;
        }

        let content = std::fs::read_to_string(entry.path())?;
        let kind = extract_field(&content, "kind").unwrap_or_else(|| "fact".to_string());
        let text = extract_field(&content, "content").unwrap_or_else(|| content.clone());

        let url = format!("{}/api/v1/projects/{}/memories", base_url, project_id);
        let resp = client
            .post(&url)
            .bearer_auth(&token)
            .json(&serde_json::json!({
                "kind": kind,
                "content": text,
            }))
            .send()
            .await?;

        if resp.status().is_success() {
            synced += 1;
        } else {
            skipped += 1;
        }
    }

    println!("{} Sync complete: {} synced, {} skipped", console::style("✓").green(), synced, skipped);

    Ok(())
}

async fn import_memory(file: &str) -> Result<()> {
    let content = std::fs::read_to_string(file)?;
    let memory_dir = ProjectConfig::memory_dir()?;
    std::fs::create_dir_all(&memory_dir)?;

    let id = uuid::Uuid::new_v4().to_string();
    let filename = format!("{}.yaml", &id[..8]);
    let filepath = memory_dir.join(&filename);

    let memory = serde_yaml::to_string(&serde_json::json!({
        "id": id,
        "kind": "context",
        "content": content,
        "source": "import",
        "created_at": chrono::Utc::now().to_rfc3339(),
    }))?;

    std::fs::write(&filepath, memory)?;
    println!("{} Imported to {}", console::style("✓").green(), filename);

    Ok(())
}

fn extract_field(content: &str, field: &str) -> Option<String> {
    for line in content.lines() {
        let line = line.trim();
        if let Some(rest) = line.strip_prefix(&format!("{}:", field)) {
            return Some(rest.trim().trim_matches('"').to_string());
        }
    }
    None
}
