use anyhow::Result;
use std::path::Path;

/// Get directory tree structure
pub fn get_tree(path: &Path, depth: usize) -> Result<String> {
    let mut output = String::new();
    build_tree(path, path, depth, 0, &mut output)?;
    Ok(output)
}

fn build_tree(base: &Path, dir: &Path, max_depth: usize, current_depth: usize, output: &mut String) -> Result<()> {
    if current_depth > max_depth {
        return Ok(());
    }

    let mut entries: Vec<_> = std::fs::read_dir(dir)?
        .filter_map(|e| e.ok())
        .collect();

    entries.sort_by_key(|e| e.file_name());

    for entry in entries {
        let name = entry.file_name().to_string_lossy().to_string();
        if name.starts_with('.') || name == "node_modules" || name == "target" {
            continue;
        }

        let indent = "  ".repeat(current_depth);
        let path = entry.path();

        if path.is_dir() {
            output.push_str(&format!("{}{}/\n", indent, name));
            build_tree(base, &path, max_depth, current_depth + 1, output)?;
        } else {
            output.push_str(&format!("{}{}\n", indent, name));
        }
    }

    Ok(())
}
