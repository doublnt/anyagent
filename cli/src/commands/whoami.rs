use anyhow::Result;
use crate::config::GlobalConfig;

pub async fn run() -> Result<()> {
    let config = GlobalConfig::load()?;

    if !config.is_authenticated() {
        println!("Not logged in. Run {} to authenticate.", console::style("agentx login").cyan());
        return Ok(());
    }

    println!("API:   {}", config.api_base_url());
    if let Some(email) = &config.email {
        println!("Email: {}", email);
    }
    if let Some(user_id) = &config.user_id {
        println!("User:  {}", user_id);
    }
    // TODO: Fetch user info from API

    Ok(())
}
