//! Implements the `restack setup` command.

use std::io::{self, Write};
use std::os::unix::fs::OpenOptionsExt;
use std::{env, fs, path};

use anyhow::{Context, Result};

use crate::git::{self, Git};

const USAGE: &str = "\
USAGE:
    restack setup [OPTIONS]

Configures Git to use restack during an interactive rebase.
If you prefer to configure Git manually, see,
    https://github.com/abhinav/restack#manual-setup
If you want restack to run on an opt-in basis, see,
    https://github.com/abhinav/restack#can-i-make-restacking-opt-in

OPTIONS:
    -h, --help
            Print help information.

        --print-edit-script
            Print the shell script without setting it up.

            This shell script is used as the editor for interactive rebases.
            It invokes 'restack
            edit' on the rebase instructions.
";

/// Arguments for the "restack setup" command.
#[derive(Debug, PartialEq, Eq)]
struct Args {
    /// If set, the shell script will be printed instead of being installed.
    print_script: bool,
}

/// Shell script to run as the sequence editor.
static EDIT_SCRIPT: &[u8] = include_bytes!("edit.sh");

/// Runs the `restack setup` command.
pub fn run(mut parser: lexopt::Parser) -> Result<()> {
    let args = {
        let mut args = Args {
            print_script: false,
        };

        while let Some(arg) = parser.next()? {
            match arg {
                lexopt::Arg::Long("print-edit-script") => {
                    args.print_script = true;
                },
                lexopt::Arg::Short('h') | lexopt::Arg::Long("help") => {
                    eprint!("{}", USAGE);
                    return Ok(());
                },
                _ => return Err(arg.unexpected().into()),
            }
        }

        args
    };

    if args.print_script {
        return io::stdout()
            .write_all(EDIT_SCRIPT)
            .context("Could nto print edit script");
    }

    let home =
        path::PathBuf::from(env::var("HOME").context("Could not determine $HOME is not defined")?);

    let edit_path = {
        // TODO: Consider using xdg-home instead.
        let mut path = home.join(".restack");
        fs::create_dir_all(&path).context("Unable to create $HOME/.restack")?;

        path.push("edit.sh");
        fs::OpenOptions::new()
            .create(true)
            .write(true)
            .mode(0o755)
            .open(&path)
            .and_then(|mut f| f.write_all(EDIT_SCRIPT))
            .context("Failed to write .restack/edit.sh")?;

        path
    };

    let git_shell = git::Shell::new();
    git_shell
        .set_global_config_str("sequence.editor", &edit_path)
        .context("Could not update sequence.editor")?;

    eprintln!("restack has been successfully set up");

    Ok(())
}
