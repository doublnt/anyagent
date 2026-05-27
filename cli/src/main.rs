mod commands;
mod config;
mod api;
mod runner;
mod utils;

use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "agentx", version, about = "AnyAgent - Agent enhancement platform for developers")]
struct Cli {
    #[command(subcommand)]
    command: Commands,

    /// Enable verbose logging
    #[arg(long, global = true)]
    verbose: bool,
}

#[derive(Subcommand)]
enum Commands {
    /// Initialize .agentx/ in current directory
    Init(commands::init::InitArgs),

    /// Login to AnyAgent
    Login(commands::login::LoginArgs),

    /// Logout
    Logout,

    /// Show current user and subscription status
    Whoami,

    /// Show project status
    Status,

    /// Search Agent Store
    Search(commands::search::SearchArgs),

    /// Install an agent pack
    Install(commands::install::InstallArgs),

    /// Uninstall an agent pack
    Uninstall(commands::uninstall::UninstallArgs),

    /// List installed agents
    List,

    /// Update installed agents
    Update(commands::update::UpdateArgs),

    /// Run a task with an agent
    Run(commands::run::RunArgs),

    /// Start MCP server
    Mcp(commands::mcp::McpArgs),

    /// Memory management
    #[command(subcommand)]
    Memory(commands::memory::MemoryCommands),

    /// Trace management
    #[command(subcommand)]
    Trace(commands::trace::TraceCommands),

    /// Eval management
    #[command(subcommand)]
    Eval(commands::eval::EvalCommands),

    /// Configuration management
    #[command(subcommand)]
    Config(commands::config::ConfigCommands),
}

#[tokio::main]
async fn main() {
    let cli = Cli::parse();

    let level = if cli.verbose {
        tracing::level_filters::LevelFilter::DEBUG
    } else {
        tracing::level_filters::LevelFilter::WARN
    };
    tracing_subscriber::fmt().with_max_level(level).init();

    let result = match cli.command {
        Commands::Init(args) => commands::init::run(args).await,
        Commands::Login(args) => commands::login::run(args).await,
        Commands::Logout => commands::logout::run().await,
        Commands::Whoami => commands::whoami::run().await,
        Commands::Status => commands::status::run().await,
        Commands::Search(args) => commands::search::run(args).await,
        Commands::Install(args) => commands::install::run(args).await,
        Commands::Uninstall(args) => commands::uninstall::run(args).await,
        Commands::List => commands::list::run().await,
        Commands::Update(args) => commands::update::run(args).await,
        Commands::Run(args) => commands::run::run(args).await,
        Commands::Mcp(args) => commands::mcp::run(args).await,
        Commands::Memory(cmd) => commands::memory::run(cmd).await,
        Commands::Trace(cmd) => commands::trace::run(cmd).await,
        Commands::Eval(cmd) => commands::eval::run(cmd).await,
        Commands::Config(cmd) => commands::config::run(cmd).await,
    };

    if let Err(e) = result {
        eprintln!("{} {}", console::style("error:").red().bold(), e);
        std::process::exit(1);
    }
}
