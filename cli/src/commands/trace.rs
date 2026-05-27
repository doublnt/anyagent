use anyhow::{Context, Result};
use clap::Subcommand;
use crate::config::{project::ProjectConfig, GlobalConfig};

#[derive(Subcommand)]
pub enum TraceCommands {
    /// List recent traces
    List {
        /// Max results
        #[arg(long, default_value = "10")]
        limit: u32,

        /// List from cloud
        #[arg(long)]
        cloud: bool,
    },

    /// Show trace details
    Show {
        /// Trace ID (or prefix)
        id: String,
    },

    /// Export trace
    Export {
        /// Trace ID
        id: String,

        /// Output format
        #[arg(long, default_value = "json")]
        format: String,
    },

    /// Sync local traces to cloud
    Sync,
}

pub async fn run(cmd: TraceCommands) -> Result<()> {
    match cmd {
        TraceCommands::List { limit, cloud } => {
            if cloud {
                list_cloud_traces(limit).await?;
            } else {
                list_local_traces(limit)?;
            }
        }
        TraceCommands::Show { id } => {
            show_trace(&id)?;
        }
        TraceCommands::Export { id, format } => {
            export_trace(&id, &format)?;
        }
        TraceCommands::Sync => {
            sync_traces().await?;
        }
    }

    Ok(())
}

fn list_local_traces(limit: u32) -> Result<()> {
    let trace_dir = ProjectConfig::traces_dir()?;
    if !trace_dir.exists() {
        println!("No traces yet.");
        println!("Run {} to create one.", console::style("agentx run <task>").cyan());
        return Ok(());
    }

    let mut entries: Vec<_> = std::fs::read_dir(&trace_dir)?
        .filter_map(|e| e.ok())
        .filter(|e| e.path().extension().map_or(false, |ext| ext == "json"))
        .collect();

    entries.sort_by_key(|e| std::cmp::Reverse(e.metadata().ok().and_then(|m| m.modified().ok())));

    if entries.is_empty() {
        println!("No traces yet.");
        return Ok(());
    }

    println!("Recent traces:");
    println!();
    for entry in entries.iter().take(limit as usize) {
        let content = std::fs::read_to_string(entry.path())?;
        let trace: serde_json::Value = serde_json::from_str(&content)?;

        let id = trace["id"].as_str().unwrap_or("?");
        let short_id = &id[..8.min(id.len())];
        let status = trace["status"].as_str().unwrap_or("unknown");
        let task = trace["task"].as_str().unwrap_or("no task");

        let status_style = match status {
            "completed" => console::style(status).green(),
            "failed" => console::style(status).red(),
            _ => console::style(status).yellow(),
        };

        println!("  {} {} {}", console::style(short_id).dim(), status_style, task);
    }

    println!();
    println!("Run {} to see details.", console::style("agentx trace show <id>").cyan());

    Ok(())
}

async fn list_cloud_traces(limit: u32) -> Result<()> {
    let config = GlobalConfig::load()?;
    if !config.is_authenticated() {
        anyhow::bail!("Not logged in. Run `agentx login` first.");
    }

    let project_config = ProjectConfig::load()?;
    let project_id = project_config.project_id.as_deref().unwrap_or("default");

    let client = reqwest::Client::new();
    let url = format!(
        "{}/api/v1/projects/{}/traces?limit={}",
        config.api_base_url(), project_id, limit
    );

    let resp = client
        .get(&url)
        .bearer_auth(config.token.as_ref().unwrap())
        .send()
        .await
        .context("Failed to connect to API")?;

    if !resp.status().is_success() {
        anyhow::bail!("API error: {}", resp.status());
    }

    let traces: Vec<serde_json::Value> = resp.json().await?;

    if traces.is_empty() {
        println!("No cloud traces found.");
        return Ok(());
    }

    println!("Cloud traces:");
    println!();
    for t in &traces {
        let id = t["id"].as_str().unwrap_or("?");
        let short_id = &id[..8.min(id.len())];
        let status = t["status"].as_str().unwrap_or("unknown");
        let task = t["task_description"].as_str().unwrap_or("no task");

        let status_style = match status {
            "completed" => console::style(status).green(),
            "failed" => console::style(status).red(),
            _ => console::style(status).yellow(),
        };

        println!("  {} {} {}", console::style(short_id).dim(), status_style, task);
    }

    Ok(())
}

fn show_trace(id_prefix: &str) -> Result<()> {
    let trace_dir = ProjectConfig::traces_dir()?;
    if !trace_dir.exists() {
        anyhow::bail!("No traces directory found.");
    }

    // Find trace by ID prefix
    for entry in std::fs::read_dir(&trace_dir)? {
        let entry = entry?;
        let filename = entry.file_name().to_string_lossy().to_string();
        if !filename.ends_with(".json") {
            continue;
        }

        let content = std::fs::read_to_string(entry.path())?;
        let trace: serde_json::Value = serde_json::from_str(&content)?;
        let trace_id = trace["id"].as_str().unwrap_or("");

        if trace_id.starts_with(id_prefix) || filename.contains(id_prefix) {
            println!("{}", serde_json::to_string_pretty(&trace)?);
            return Ok(());
        }
    }

    anyhow::bail!("Trace not found: {}", id_prefix)
}

fn export_trace(id_prefix: &str, format: &str) -> Result<()> {
    let trace_dir = ProjectConfig::traces_dir()?;
    if !trace_dir.exists() {
        anyhow::bail!("No traces directory found.");
    }

    for entry in std::fs::read_dir(&trace_dir)? {
        let entry = entry?;
        let filename = entry.file_name().to_string_lossy().to_string();
        if !filename.ends_with(".json") {
            continue;
        }

        let content = std::fs::read_to_string(entry.path())?;
        let trace: serde_json::Value = serde_json::from_str(&content)?;
        let trace_id = trace["id"].as_str().unwrap_or("");

        if trace_id.starts_with(id_prefix) || filename.contains(id_prefix) {
            match format {
                "json" => {
                    println!("{}", serde_json::to_string_pretty(&trace)?);
                }
                "yaml" => {
                    println!("{}", serde_yaml::to_string(&trace)?);
                }
                _ => {
                    anyhow::bail!("Unsupported format: {}", format);
                }
            }
            return Ok(());
        }
    }

    anyhow::bail!("Trace not found: {}", id_prefix)
}

async fn sync_traces() -> Result<()> {
    let config = GlobalConfig::load()?;
    if !config.is_authenticated() {
        anyhow::bail!("Not logged in. Run `agentx login` first.");
    }

    let trace_dir = ProjectConfig::traces_dir()?;
    if !trace_dir.exists() {
        println!("No local traces to sync.");
        return Ok(());
    }

    let project_config = ProjectConfig::load()?;
    let project_id = project_config.project_id.as_deref().unwrap_or("default");

    let client = reqwest::Client::new();
    let base_url = config.api_base_url().to_string();
    let token = config.token.as_ref().unwrap().clone();

    let mut synced = 0;
    let mut skipped = 0;

    for entry in std::fs::read_dir(&trace_dir)? {
        let entry = entry?;
        if !entry.path().extension().map_or(false, |e| e == "json") {
            continue;
        }

        let content = std::fs::read_to_string(entry.path())?;
        let trace: serde_json::Value = serde_json::from_str(&content)?;

        let task = trace["task"].as_str().unwrap_or("").to_string();
        let agent = trace["agent"].as_str().map(|s| s.to_string());
        let status = trace["status"].as_str().unwrap_or("completed");

        // Create trace on cloud
        let url = format!("{}/api/v1/projects/{}/traces", base_url, project_id);
        let resp = client
            .post(&url)
            .bearer_auth(&token)
            .json(&serde_json::json!({
                "task": task,
                "agent_name": agent,
            }))
            .send()
            .await?;

        if resp.status().is_success() {
            let created: serde_json::Value = resp.json().await?;
            let trace_id = created["id"].as_str().unwrap_or("");

            // Complete the trace
            if status != "running" {
                let complete_url = format!("{}/api/v1/traces/{}/complete", base_url, trace_id);
                client
                    .post(&complete_url)
                    .bearer_auth(&token)
                    .json(&serde_json::json!({
                        "status": status,
                    }))
                    .send()
                    .await?;
            }

            synced += 1;
        } else {
            skipped += 1;
        }
    }

    println!("{} Sync complete: {} synced, {} skipped", console::style("✓").green(), synced, skipped);

    Ok(())
}
