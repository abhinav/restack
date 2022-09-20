//! Implements the `restack setup` command.

use std::{
    fs,
    io::{self, Write},
    os::unix::fs::OpenOptionsExt,
};

use anyhow::{anyhow, Context, Result};
use argh::FromArgs;

use crate::git::{self, Git};

#[derive(Debug, PartialEq, Eq, FromArgs)]
/// sets up restack
#[argh(subcommand, name = "setup")]
pub struct Args {
    /// print the editor shell script
    #[argh(switch, long = "print-edit-script")]
    print_script: bool,
}

/// Shell script to run as the sequence editor.
static EDIT_SCRIPT: &[u8] = include_bytes!("edit.sh");

/// Runs the `restack setup` command.
pub fn run(args: &Args) -> Result<()> {
    if args.print_script {
        return io::stdout()
            .write_all(EDIT_SCRIPT)
            .context("Could nto print edit script");
    }

    let home = dirs::home_dir().ok_or_else(|| anyhow!("Home directory not found"))?;

    let edit_path = {
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
