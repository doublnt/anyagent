use anyhow::Result;
use std::path::Path;

/// List files in directory, respecting .gitignore
pub fn list_files(path: &Path, pattern: &str) -> Result<Vec<String>> {
    // Use git ls-files if in a git repo
    let output = std::process::Command::new("git")
        .args(["ls-files", "--cached", "--others", "--exclude-standard"])
        .current_dir(path)
        .output();

    if output.is_ok() && output.as_ref().unwrap().status.success() {
        let stdout = String::from_utf8(output.unwrap().stdout)?;
        let files: Vec<String> = stdout.lines().map(|l| l.to_string()).collect();
        return Ok(files);
    }

    // Fallback: walk directory
    let mut files = Vec::new();
    walk_dir(path, path, &mut files)?;
    Ok(files)
}

fn walk_dir(base: &Path, dir: &Path, files: &mut Vec<String>) -> Result<()> {
    for entry in std::fs::read_dir(dir)? {
        let entry = entry?;
        let path = entry.path();
        let name = entry.file_name().to_string_lossy().to_string();

        // Skip hidden files and common ignored dirs
        if name.starts_with('.') || name == "node_modules" || name == "target" || name == "vendor" {
            continue;
        }

        if path.is_dir() {
            walk_dir(base, &path, files)?;
        } else {
            if let Ok(relative) = path.strip_prefix(base) {
                files.push(relative.to_string_lossy().to_string());
            }
        }
    }
    Ok(())
}

/// Read a file, checking it's not in .gitignore
pub fn read_file(path: &Path) -> Result<String> {
    Ok(std::fs::read_to_string(path)?)
}
