use anyhow::Result;
use std::path::Path;

pub struct GitContext {
    pub branch: Option<String>,
    pub remote_url: Option<String>,
    pub last_commit: Option<CommitInfo>,
    pub status: Vec<FileStatus>,
}

pub struct CommitInfo {
    pub hash: String,
    pub message: String,
    pub author: String,
    pub date: String,
}

pub struct FileStatus {
    pub path: String,
    pub status: String, // M, A, D, ??
}

impl GitContext {
    pub fn collect(repo_path: &Path) -> Result<Self> {
        Ok(Self {
            branch: get_branch(repo_path).ok(),
            remote_url: get_remote_url(repo_path).ok(),
            last_commit: get_last_commit(repo_path).ok(),
            status: get_status(repo_path).unwrap_or_default(),
        })
    }
}

fn get_branch(path: &Path) -> Result<String> {
    let output = std::process::Command::new("git")
        .args(["rev-parse", "--abbrev-ref", "HEAD"])
        .current_dir(path)
        .output()?;
    Ok(String::from_utf8(output.stdout)?.trim().to_string())
}

fn get_remote_url(path: &Path) -> Result<String> {
    let output = std::process::Command::new("git")
        .args(["remote", "get-url", "origin"])
        .current_dir(path)
        .output()?;
    Ok(String::from_utf8(output.stdout)?.trim().to_string())
}

fn get_last_commit(path: &Path) -> Result<CommitInfo> {
    let output = std::process::Command::new("git")
        .args(["log", "-1", "--format=%H%n%s%n%an%n%ai"])
        .current_dir(path)
        .output()?;
    let stdout = String::from_utf8(output.stdout)?;
    let parts: Vec<&str> = stdout.lines().collect();
    if parts.len() >= 4 {
        Ok(CommitInfo {
            hash: parts[0].to_string(),
            message: parts[1].to_string(),
            author: parts[2].to_string(),
            date: parts[3].to_string(),
        })
    } else {
        anyhow::bail!("Failed to parse git log")
    }
}

fn get_status(path: &Path) -> Result<Vec<FileStatus>> {
    let output = std::process::Command::new("git")
        .args(["status", "--porcelain"])
        .current_dir(path)
        .output()?;
    let stdout = String::from_utf8(output.stdout)?;
    let statuses = stdout
        .lines()
        .filter(|line| line.len() >= 3)
        .map(|line| FileStatus {
            status: line[..2].trim().to_string(),
            path: line[3..].to_string(),
        })
        .collect();
    Ok(statuses)
}
