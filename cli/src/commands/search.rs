use anyhow::{Context, Result};
use clap::Args;
use crate::api::registry::AgentInfo;
use crate::config::GlobalConfig;

#[derive(Args)]
pub struct SearchArgs {
    /// Search query
    query: String,

    /// Category filter
    #[arg(long)]
    category: Option<String>,
}

pub async fn run(args: SearchArgs) -> Result<()> {
    let config = GlobalConfig::load()?;
    let api_base = config.api_base_url().to_string();

    let mut url = format!("{}/api/v1/agents?q={}", api_base, urlencoding::encode(&args.query));
    if let Some(cat) = &args.category {
        url.push_str(&format!("&category={}", urlencoding::encode(cat)));
    }

    let client = reqwest::Client::new();
    let response = client
        .get(&url)
        .send()
        .await
        .context("Failed to connect to registry")?;

    if !response.status().is_success() {
        anyhow::bail!("Registry error: {}", response.status());
    }

    let body = response.text().await.context("Failed to read response")?;
    let agents: Vec<AgentInfo> = serde_json::from_str(&body)
        .context(format!("Failed to parse response: {}", &body[..body.len().min(200)]))?;

    if agents.is_empty() {
        println!("No agents found for: {}", args.query);
        return Ok(());
    }

    println!("Results for: {}", console::style(&args.query).cyan());
    println!();

    for agent in &agents {
        println!(
            "  {} {} ({})",
            console::style(&agent.name).green(),
            console::style(format!("v{}", agent.version)).dim(),
            agent.category.as_deref().unwrap_or("general")
        );
        if let Some(desc) = &agent.description {
            println!("    {}", desc);
        }
        println!("    Downloads: {}", agent.download_count);
        println!();
    }

    println!("Run {} to install an agent", console::style("agentx install <name>").cyan());

    Ok(())
}
