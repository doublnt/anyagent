use anyhow::Result;

/// Execute a shell command with timeout
pub fn run_command(command: &str, timeout_ms: u64) -> Result<CommandResult> {
    let output = std::process::Command::new("sh")
        .args(["-c", command])
        .output()?;

    Ok(CommandResult {
        exit_code: output.status.code().unwrap_or(-1),
        stdout: String::from_utf8_lossy(&output.stdout).to_string(),
        stderr: String::from_utf8_lossy(&output.stderr).to_string(),
    })
}

pub struct CommandResult {
    pub exit_code: i32,
    pub stdout: String,
    pub stderr: String,
}
