//! Implements the `restack setup` command.

use std::fs;
use std::io::{self, Write};
use std::os::unix::fs::OpenOptionsExt;

use anyhow::{anyhow, Context, Result};

use crate::git::{self, Git};

/// Configures Git to use restack during an interactive rebase.
///
/// If you prefer to configure Git manually, see:
/// <https://github.com/abhinav/restack#manual-setup>
///
/// If you want restack to run on an opt-in basis, see:
/// <https://github.com/abhinav/restack#can-i-make-restacking-opt-in>
#[derive(Debug, PartialEq, Eq, clap::Args)]
pub struct Args {
    /// Print the shell script without setting it up.
    ///
    /// This shell script is used as the editor for interactive rebases.
    /// It invokes 'restack edit' on the rebase instructions.
    #[clap(long = "print-edit-script")]
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
