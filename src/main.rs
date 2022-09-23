//! restack is a command line tool that sits between `git rebase -i`
//! and your editor, with the intent of making the default rebase smarter.
//!
//! It doess so by making `git rebase -i` aware of intermediate branches
//! attached to commits you're changing during a rebase.
//! If it finds commits with associated branches,
//! it introduces new instructions to the rebase instruction list
//! that will update these branches if the commits move.
//!
//! Read more in the [README][1] and the [associated blog post][2].
//!
//! [1]: https://github.com/abhinav/restack/blob/main/README.md
//! [2]: https://abhinavg.net/posts/restacking-branches/

#![warn(missing_docs)]
#![feature(byte_slice_trim_ascii)]
#![feature(exit_status_error)]
#![feature(io_error_more)]
#![feature(slice_group_by)]

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
