use anyhow::{Context, Result};
use clap::Args;
use flate2::write::GzEncoder;
use flate2::Compression;
use std::io::Write;
use std::path::Path;
use tar::Builder;
use crate::config::GlobalConfig;

#[derive(Args)]
pub struct PublishArgs {
    /// Path to the agent pack directory to publish
    path: String,

    /// Agent version (auto-detected from agent.yaml if not set)
    #[arg(long)]
    version: Option<String>,

    /// Description for this version
    #[arg(long)]
    description: Option<String>,

    /// Mark this agent as a hosted service (platform runs it in sandbox)
    #[arg(long)]
    hosted: bool,

    /// Price in cents per subscription period (e.g. 499 = $4.99/month)
    #[arg(long)]
    price: Option<i32>,
}

pub async fn run(args: PublishArgs) -> Result<()> {
    let global_config = GlobalConfig::load()?;

    if !global_config.is_authenticated() {
        anyhow::bail!("Not logged in. Run 'agentx login' first.");
    }

    let api_base = global_config.api_base_url().to_string();

    let pack_path = Path::new(&args.path);
    if !pack_path.exists() {
        anyhow::bail!("Agent pack directory not found: {}", args.path);
    }

    let agent_yaml_path = pack_path.join("agent.yaml");
    if !agent_yaml_path.exists() {
        anyhow::bail!("Missing agent.yaml in {}", args.path);
    }

    // Parse agent.yaml for name and version
    let content = std::fs::read_to_string(&agent_yaml_path)
        .context("Failed to read agent.yaml")?;

    let name = parse_yaml_field(&content, "name")
        .context("Could not parse 'name' from agent.yaml")?;
    let detected_version = parse_yaml_field(&content, "version")
        .unwrap_or_else(|| "0.1.0".to_string());
    let desc = args.description
        .or_else(|| parse_yaml_field(&content, "description"))
        .unwrap_or_default();
    let category = parse_yaml_field(&content, "category").unwrap_or_default();
    let tags = parse_yaml_field(&content, "tags")
        .map(|t| t.split(',').map(|s| s.trim().to_string()).collect::<Vec<_>>().join(","))
        .unwrap_or_default();

    let version = args.version.unwrap_or(detected_version);

    println!("{} Publishing {}@{} to registry...", console::style("●").cyan(), name, version);

    // Build tarball of the pack directory
    let tarball = build_tarball(pack_path)
        .context("Failed to create tarball")?;

    // Prepare multipart form
    let client = reqwest::Client::new();
    let form = reqwest::multipart::Part::bytes(tarball)
        .file_name(format!("{}-{}.tar.gz", name, version))
        .mime_str("application/gzip")
        .unwrap();

    let mut form = reqwest::multipart::Form::new()
        .text("name", name.clone())
        .text("version", version.clone())
        .text("description", desc)
        .text("category", category)
        .text("tags", tags)
        .text("is_hosted", args.hosted.to_string())
        .text("manifest", content)
        .part("artifact", form);

    if let Some(price) = args.price {
        form = form.text("price_cents", price.to_string());
    }

    let url = format!("{}/api/v1/agents", api_base);
    let token = global_config.token
        .as_ref()
        .context("No auth token found")?;

    let response = client
        .post(&url)
        .bearer_auth(token)
        .multipart(form)
        .send()
        .await
        .context("Failed to connect to registry")?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response.text().await.unwrap_or_default();
        anyhow::bail!("Publish failed ({}): {}", status, body);
    }

    println!("{} Published {}@{}", console::style("✓").green(), name, version);

    if args.hosted {
        println!("  {} This agent will run in the AnyAgent sandbox.", console::style("●").cyan());
        println!("  Subscribers can call it via the MCP gateway.");
    }

    Ok(())
}

fn parse_yaml_field(content: &str, field: &str) -> Option<String> {
    for line in content.lines() {
        let line = line.trim();
        if let Some(rest) = line.strip_prefix(&format!("{}:", field)) {
            return Some(rest.trim().trim_matches('"').trim_matches('\'').to_string());
        }
    }
    None
}

fn build_tarball(dir: &Path) -> Result<Vec<u8>> {
    // Build tar into an in-memory buffer first
    let mut tar_data = Vec::new();
    {
        let mut builder = Builder::new(&mut tar_data);
        walk_dir(dir, dir, &mut builder)?;
        builder.finish()?;
    }

    // Gzip the tar data
    let mut gzip_data = Vec::new();
    {
        let mut encoder = GzEncoder::new(&mut gzip_data, Compression::default());
        encoder.write_all(&tar_data)?;
        encoder.finish()?;
    }

    Ok(gzip_data)
}

fn walk_dir(base: &Path, current: &Path, builder: &mut Builder<&mut Vec<u8>>) -> Result<()> {
    for entry in std::fs::read_dir(current)? {
        let entry = entry?;
        let path = entry.path();
        let relative = path.strip_prefix(base).unwrap_or(&path);
        let relative_str = relative.to_string_lossy().replace('\\', "/");

        if relative_str == "." {
            continue;
        }

        if path.is_dir() {
            for subentry in std::fs::read_dir(&path)? {
                walk_dir(base, &subentry?.path(), builder)?;
            }
        } else {
            let mut header = tar::Header::new_gnu();
            header.set_path(&relative_str)?;
            if let Ok(meta) = entry.metadata() {
                header.set_size(meta.len());
                header.set_mode(0o644);
            }
            let mut file = std::fs::File::open(&path)?;
            builder.append(&header, &mut file)?;
        }
    }
    Ok(())
}
