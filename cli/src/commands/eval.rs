use anyhow::Result;
use clap::Subcommand;
use crate::config::project::ProjectConfig;

#[derive(Subcommand)]
pub enum EvalCommands {
    /// List eval rules
    List,

    /// Run eval rules on recent trace
    Run {
        /// Specific rule names (runs all if omitted)
        #[arg(long)]
        rules: Vec<String>,
    },

    /// Add an eval rule
    Add {
        /// Rule file path
        file: String,
    },
}

pub async fn run(cmd: EvalCommands) -> Result<()> {
    let _config = ProjectConfig::load()?;

    match cmd {
        EvalCommands::List => {
            let agentx_dir = ProjectConfig::project_dir()?;
            let eval_dir = agentx_dir.join("eval");

            if !eval_dir.exists() {
                println!("No eval rules defined.");
                println!("Run {} to add one.", console::style("agentx eval add <file>").cyan());
                return Ok(());
            }

            let entries: Vec<_> = std::fs::read_dir(&eval_dir)?
                .filter_map(|e| e.ok())
                .filter(|e| e.path().extension().map_or(false, |ext| ext == "yaml"))
                .collect();

            if entries.is_empty() {
                println!("No eval rules defined.");
                return Ok(());
            }

            println!("Eval rules:");
            for entry in entries {
                let name = entry.file_name().to_string_lossy().to_string();
                println!("  {}", console::style(&name).green());
            }
        }
        EvalCommands::Run { rules } => {
            println!("Running eval...");
            if rules.is_empty() {
                println!("  Rules: all");
            } else {
                println!("  Rules: {}", rules.join(", "));
            }
            println!();
            // TODO: Implement eval execution
            println!("{} Eval engine not yet implemented.", console::style("TODO").yellow());
        }
        EvalCommands::Add { file } => {
            let agentx_dir = ProjectConfig::project_dir()?;
            let eval_dir = agentx_dir.join("eval");
            std::fs::create_dir_all(&eval_dir)?;

            let filename = std::path::Path::new(&file)
                .file_name()
                .ok_or_else(|| anyhow::anyhow!("Invalid file path"))?;

            std::fs::copy(&file, eval_dir.join(filename))?;
            println!("{} Added eval rule: {}", console::style("✓").green(), filename.to_string_lossy());
        }
    }

    Ok(())
}
