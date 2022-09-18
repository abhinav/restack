use anyhow::Result;
use argh::FromArgs;

mod edit;
mod git;
mod restack;
mod setup;

#[cfg(test)]
mod fixscript;

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
