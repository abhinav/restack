#![feature(byte_slice_trim_ascii)]
#![feature(exit_status_error)]
#![feature(io_error_more)]

use anyhow::Result;
use clap::Parser;

mod edit;
mod git;
mod io;
mod restack;
mod setup;

#[cfg(test)]
mod gitscript;

#[derive(Debug, clap::Subcommand)]
enum Command {
    Setup(setup::Args),
    Edit(edit::Args),
}

#[derive(Debug, clap::Parser)]
#[clap(author, version, about)]
struct Args {
    #[clap(subcommand)]
    cmd: Command,
}

fn main() -> Result<()> {
    let args: Args = Args::parse();
    match args.cmd {
        Command::Setup(args) => setup::run(&args),
        Command::Edit(args) => edit::run(&args),
    }
}
