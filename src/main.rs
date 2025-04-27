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

use anyhow::{Result, anyhow, bail};

mod edit;
mod exec;
mod git;
mod io;
mod restack;
mod setup;

#[cfg(test)]
mod gitscript;

const USAGE: &str = "\
USAGE:
    restack <SUBCOMMAND>

Teaches git rebase --interactive about your branches.

OPTIONS:
    -h, --help       Print help information.
    -V, --version    Print version information.

SUBCOMMANDS:
    edit     Edits the provided rebase instruction list
    setup    Configures Git to use restack during an interactive rebase
";

fn main() -> Result<()> {
    let mut parser = lexopt::Parser::from_env();
    let Some(arg) = parser.next()? else {
        bail!("Please provide a subcommand. See restack --help for more information.");
    };

    match arg {
        lexopt::Arg::Short('h') | lexopt::Arg::Long("help") => {
            eprint!("{}", USAGE);
            Ok(())
        },
        lexopt::Arg::Short('V') | lexopt::Arg::Long("version") => {
            println!("restack {}", env!("CARGO_PKG_VERSION"));
            println!("Copyright (C) 2023 Abhinav Gupta");
            println!("  <https://github.com/abhinav/restack>");
            println!("restack comes with ABSOLUTELY NO WARRANTY.");
            println!("This is free software, and you are welcome to redistribute it");
            println!("under certain conditions. See source for details.");
            Ok(())
        },
        lexopt::Arg::Value(ref cmd) => {
            let cmd = cmd
                .to_str()
                .ok_or_else(|| anyhow!("{:?} is not a valid unicode string", cmd))?;
            match cmd {
                "edit" => edit::run(parser),
                "setup" => setup::run(parser),
                _ => {
                    bail!("Unrecognized command '{}'", cmd);
                },
            }
        },
        _ => Err(arg.unexpected().into()),
    }
}
