use anyhow::{Context, Result};
use clap::Args;
use crate::config::{project::{InstalledAgent, ProjectConfig}, GlobalConfig};

#[derive(Args)]
pub struct InstallArgs {
    /// Agent name (with optional @version)
    agent: String,

    /// Force reinstall even if already installed
    #[arg(long)]
    force: bool,
}

pub async fn run(args: InstallArgs) -> Result<()> {
    let mut config = ProjectConfig::load()?;

    // Parse name@version
    let (name, version) = if let Some((n, v)) = args.agent.split_once('@') {
        (n.to_string(), Some(v.to_string()))
    } else {
        (args.agent.clone(), None)
    };

    if config.has_agent(&name) && !args.force {
        println!("Agent {} is already installed. Use --force to reinstall.", name);
        return Ok(());
    }

    let global_config = GlobalConfig::load()?;
    let api_base = global_config.api_base_url().to_string();
    let version_str = version.as_deref().unwrap_or("latest");

    // Download agent pack from registry
    println!("Downloading {}@{}...", console::style(&name).cyan(), version_str);

    let client = reqwest::Client::new();
    let download_url = format!(
        "{}/api/v1/agents/{}/{}/download",
        api_base, name, version_str
    );

    let response = client
        .get(&download_url)
        .send()
        .await
        .context("Failed to connect to registry")?;

    if !response.status().is_success() {
        anyhow::bail!(
            "Failed to download agent: {} {}",
            response.status(),
            response.text().await.unwrap_or_default()
        );
    }

    // Get the tarball bytes
    let bytes = response.bytes().await.context("Failed to download agent pack")?;

    // Extract to .agentx/agents/<name>/
    let agents_dir = ProjectConfig::agents_dir()?.join(&name);
    if agents_dir.exists() {
        std::fs::remove_dir_all(&agents_dir)?;
    }
    std::fs::create_dir_all(&agents_dir)?;

    extract_tarball(&bytes, &agents_dir)?;

    // Load manifest to get version info
    let manifest_path = agents_dir.join("agent.yaml");
    let installed_version = if manifest_path.exists() {
        let content = std::fs::read_to_string(&manifest_path)?;
        parse_version_from_yaml(&content).unwrap_or_else(|| version_str.to_string())
    } else {
        version_str.to_string()
    };

    // Update config
    if let Some(idx) = config.agents.iter().position(|a| a.name == name) {
        config.agents[idx].version = installed_version.clone();
        config.agents[idx].installed_at = chrono::Utc::now().to_rfc3339();
    } else {
        config.agents.push(InstalledAgent {
            name: name.clone(),
            version: installed_version.clone(),
            installed_at: chrono::Utc::now().to_rfc3339(),
        });
    }
    config.save()?;

    println!(
        "{} Installed {}@{}",
        console::style("✓").green(),
        console::style(&name).cyan(),
        installed_version
    );

    // Show what was installed
    if manifest_path.exists() {
        let content = std::fs::read_to_string(&manifest_path)?;
        if let Some(desc) = parse_description_from_yaml(&content) {
            println!("  {}", console::style(desc).dim());
        }
    }

    Ok(())
}

fn extract_tarball(data: &[u8], dest: &std::path::Path) -> Result<()> {
    use flate2::read::GzDecoder;
    use tar::Archive;

    let decoder = GzDecoder::new(data);
    let mut archive = Archive::new(decoder);

    // Unpack to a temp directory first, then move contents
    let temp_dir = tempfile::tempdir()?;
    archive.unpack(temp_dir.path())?;

    // Find the actual content (might be nested in a subdirectory)
    let entries: Vec<_> = std::fs::read_dir(temp_dir.path())?
        .filter_map(|e| e.ok())
        .collect();

    if entries.len() == 1 && entries[0].path().is_dir() {
        // Single directory - move its contents
        copy_dir_all(&entries[0].path(), dest)?;
    } else {
        // Multiple items - copy all
        copy_dir_all(temp_dir.path(), dest)?;
    }

    Ok(())
}

fn copy_dir_all(src: &std::path::Path, dst: &std::path::Path) -> Result<()> {
    std::fs::create_dir_all(dst)?;
    for entry in std::fs::read_dir(src)? {
        let entry = entry?;
        let ty = entry.file_type()?;
        if ty.is_dir() {
            copy_dir_all(&entry.path(), &dst.join(entry.file_name()))?;
        } else {
            std::fs::copy(entry.path(), dst.join(entry.file_name()))?;
        }
    }
    Ok(())
}

fn parse_version_from_yaml(content: &str) -> Option<String> {
    for line in content.lines() {
        let line = line.trim();
        if let Some(rest) = line.strip_prefix("version:") {
            return Some(rest.trim().to_string());
        }
    }
    None
}

fn parse_description_from_yaml(content: &str) -> Option<String> {
    for line in content.lines() {
        let line = line.trim();
        if let Some(rest) = line.strip_prefix("description:") {
            return Some(rest.trim().trim_matches('"').to_string());
        }
    }
    None
}
