use anyhow::Result;
use clap::Args;
use crate::config::GlobalConfig;

#[derive(Args)]
pub struct LoginArgs {
    /// Login with a token directly (for CI environments)
    #[arg(long)]
    token: Option<String>,

    /// API base URL override
    #[arg(long)]
    api_url: Option<String>,
}

pub async fn run(args: LoginArgs) -> Result<()> {
    let mut config = GlobalConfig::load()?;

    if let Some(api_url) = args.api_url {
        config.api_url = Some(api_url);
    }

    if let Some(token) = args.token {
        // Direct token login (CI mode)
        config.token = Some(token);
        config.save()?;
        println!("{} Logged in with token", console::style("✓").green());
        return Ok(());
    }

    // Interactive login - open browser for OAuth
    let api_base = config.api_base_url().to_string();
    let auth_url = format!("{}/auth/github", api_base);

    println!("Opening browser for authentication...");
    println!("If the browser doesn't open, visit:");
    println!("  {}", console::style(&auth_url).cyan());
    println!();

    // TODO: Start local callback server, open browser, receive token
    // For now, prompt for manual token entry
    println!("After authenticating, paste your token:");
    let token = dialoguer::Password::new()
        .with_prompt("Token")
        .interact()?;

    if token.trim().is_empty() {
        anyhow::bail!("No token provided");
    }

    config.token = Some(token.trim().to_string());

    // TODO: Verify token with API and get user info
    // let client = ApiClient::new()?.with_token(token.trim().to_string());
    // let resp = client.get("/api/v1/auth/me").await?;

    config.save()?;

    println!("{} Logged in successfully", console::style("✓").green());

    Ok(())
}
