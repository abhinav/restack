use anyhow::Result;
use argh::FromArgs;

pub mod edit;
pub mod git;
pub mod restack;
pub mod setup;

#[cfg(test)]
pub mod fixscript;

#[derive(Debug, FromArgs)]
#[argh(subcommand)]
enum Command {
    Setup(setup::Args),
    Edit(edit::Args),
}

#[derive(Debug, FromArgs)]
/// restack makes git rebase --interactive nicer.
struct Args {
    #[argh(subcommand)]
    cmd: Command,
}

fn main() -> Result<()> {
    let args: Args = argh::from_env();
    match args.cmd {
        Command::Setup(args) => setup::run(&args),
        Command::Edit(args) => edit::run(&args),
    }
}
