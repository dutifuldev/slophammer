use clap::{Parser, Subcommand, ValueEnum};
use slophammer_app::{AppResult, CheckOptions, DirectOptions, OutputFormat};
use std::io::{self, Write};
use std::process::ExitCode;

#[derive(Parser)]
#[command(name = "slophammer-rs")]
#[command(about = "Rust implementation of the Slophammer repository quality checker.")]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    Check {
        path: String,
        #[arg(long, value_enum, default_value_t = FormatArg::Text)]
        format: FormatArg,
        #[arg(long)]
        execute: bool,
        #[arg(long = "only")]
        only_rule_ids: Vec<String>,
    },
    Explain {
        rule_id: String,
    },
    Rules {
        #[arg(long, value_enum, default_value_t = FormatArg::Text)]
        format: FormatArg,
    },
    Dry {
        #[arg(default_value = ".")]
        path: String,
        #[arg(long, value_enum, default_value_t = FormatArg::Text)]
        format: FormatArg,
        #[arg(long = "max-findings")]
        max_findings: Option<usize>,
    },
    Boundaries {
        #[arg(default_value = ".")]
        path: String,
        #[arg(long, value_enum, default_value_t = FormatArg::Text)]
        format: FormatArg,
    },
    Unsafe {
        #[arg(default_value = ".")]
        path: String,
        #[arg(long, value_enum, default_value_t = FormatArg::Text)]
        format: FormatArg,
    },
}

#[derive(Clone, Copy, Debug, ValueEnum)]
enum FormatArg {
    Text,
    Json,
    Sarif,
}

fn main() -> ExitCode {
    let cli = Cli::parse();
    exit(run(cli.command))
}

fn run(command: Command) -> AppResult {
    match command {
        Command::Check {
            path,
            format,
            execute,
            only_rule_ids,
        } => slophammer_app::check(CheckOptions {
            root: path,
            format: format.into(),
            execute,
            only_rule_ids,
        }),
        Command::Explain { rule_id } => slophammer_app::explain(&rule_id),
        Command::Rules { format } => slophammer_app::rules(format.into()),
        Command::Dry {
            path,
            format,
            max_findings,
        } => slophammer_app::dry(DirectOptions {
            root: path,
            format: format.into(),
            max_findings,
        }),
        Command::Boundaries { path, format } => slophammer_app::boundaries(DirectOptions {
            root: path,
            format: format.into(),
            max_findings: None,
        }),
        Command::Unsafe { path, format } => slophammer_app::unsafe_policy(DirectOptions {
            root: path,
            format: format.into(),
            max_findings: None,
        }),
    }
}

fn exit(result: AppResult) -> ExitCode {
    let mut stdout = io::stdout();
    let mut stderr = io::stderr();
    let _ = stdout.write_all(result.stdout.as_bytes());
    let _ = stderr.write_all(result.stderr.as_bytes());
    ExitCode::from(u8::try_from(result.code).unwrap_or(2))
}

impl From<FormatArg> for OutputFormat {
    fn from(value: FormatArg) -> Self {
        match value {
            FormatArg::Text => Self::Text,
            FormatArg::Json => Self::Json,
            FormatArg::Sarif => Self::Sarif,
        }
    }
}
