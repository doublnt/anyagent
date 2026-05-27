use anyhow::Result;
use crate::config::GlobalConfig;

pub async fn run() -> Result<()> {
    let mut config = GlobalConfig::load()?;
    config.token = None;
    config.user_id = None;
    config.save()?;

    println!("{} Logged out", console::style("✓").green());
    Ok(())
}
