//! Implements the `restack edit` command.

use std::borrow::Cow;
use std::{env, fs, path, process};

use anyhow::{bail, Context, Result};

use crate::{git, restack};

/// Edits the provided rebase instruction list.
///
/// This augments the rebase instruction list,
/// adding commands to move affected branches in the stack during the rebase.
///
/// Set up Git to use this command as the sequence.editor.
/// See <https://github.com/abhinav/restack#setup>
#[derive(Debug, PartialEq, Eq, clap::Args)]
pub struct Args {
    /// Editor to use for rebase instructions.
    ///
    /// Defaults to $EDITOR.
    #[clap(short = 'e', long = "editor")]
    editor: Option<String>,

    /// Path to the rebase instruction list.
    #[clap(value_name = "FILE")]
    file: path::PathBuf,
}

/// Runs the `restack edit` command.
pub fn run(args: &Args) -> Result<()> {
    let cwd = env::current_dir().context("Could not determine current working directory")?;
    let temp_dir = tempfile::tempdir().context("Failed to create temporary directory")?;

    // TODO: Check core.editor, GIT_EDITOR, and then EDITOR.
    let editor: Cow<str> = match args.editor.as_ref() {
        Some(s) if !s.is_empty() => Cow::Borrowed(s),
        _ => match env::var("EDITOR") {
            Ok(s) if !s.is_empty() => Cow::Owned(s),
            Err(env::VarError::NotPresent) => Cow::Borrowed("vim"),
            Err(err) => return Err(err).context("Unable to look up EDITOR"),
            _ => {
                bail!("No editor specified: please use --editor or set EDITOR")
            },
        },
    };

    let git_shell = git::Shell::new();

    // The file should be named git-rebase-todo to make file-type detection
    // in different editors work correctly.
    let outfile_path = temp_dir.path().join("git-rebase-todo");
    {
        let infile = fs::File::open(&args.file).context("Failed while reading git-rebase-todo")?;
        let outfile =
            fs::File::create(&outfile_path).context("Failed to create new git-rebase-todo")?;
        let cfg = restack::Config::new(&cwd, git_shell);
        // TODO: determine remote
        cfg.restack(Some("origin"), infile, outfile)?;
    };

    // GIT_EDITOR/EDITOR can be any shell command, including FOO=bar $some_editor.
    // So we need to use `sh` to interpret it instead of executing it directly.
    // We baically run,
    //   sh -c "$GIT_EDITOR $1" "restack" $FILE
    process::Command::new("sh")
        .arg("-c")
        .arg(format!("{} \"$1\"", editor))
        .arg("restack")
        .arg(&outfile_path)
        .status()
        .context("Could not run EDITOR")?
        .exit_ok()
        .context("Editor returned non-zero status")?;

    crate::io::rename(&outfile_path, &args.file).with_context(|| {
        format!(
            "Could not overwrite {} with {}",
            &args.file.display(),
            &outfile_path.display()
        )
    })
}
