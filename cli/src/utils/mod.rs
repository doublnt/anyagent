use std::path::Path;

/// Check if a directory is inside a git repository
pub fn is_git_repo(path: &Path) -> bool {
    path.join(".git").exists() || {
        let mut dir = path.to_path_buf();
        loop {
            if dir.join(".git").exists() {
                return true;
            }
            if !dir.pop() {
                return false;
            }
        }
    }
}

/// Get the git repo root
pub fn git_root(path: &Path) -> Option<std::path::PathBuf> {
    let mut dir = path.to_path_buf();
    loop {
        if dir.join(".git").exists() {
            return Some(dir);
        }
        if !dir.pop() {
            return None;
        }
    }
}

/// Truncate a string to max_len chars, appending "..." if truncated
pub fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len.saturating_sub(3)])
    }
}
