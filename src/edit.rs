//! Implements the `restack edit` command.

use std::{borrow::Cow, env, fs, path, process};

use anyhow::{bail, Context, Result};
use argh::FromArgs;

use crate::restack;

/// edits the instruction list for an interactive rebase
#[derive(Debug, PartialEq, Eq, FromArgs)]
#[argh(subcommand, name = "edit")]
pub struct Args {
    /// editor for rebase instructions
    #[argh(option, short = 'e', long = "editor")]
    editor: Option<String>,

    #[argh(positional, arg_name = "FILE")]
    /// file to edit
    file: path::PathBuf,
}

/// Runs the `restack edit` command.
pub fn run(args: &Args) -> Result<()> {
    let temp_dir = tempfile::tempdir().context("create temporary directory")?;

    let editor: Cow<str> = match args.editor.as_ref() {
        Some(s) if !s.is_empty() => Cow::Borrowed(s),
        _ => match env::var("EDITOR") {
            Ok(s) if !s.is_empty() => Cow::Owned(s),
            Err(env::VarError::NotPresent) => Cow::Borrowed("vim"),
            Err(err) => return Err(err).context("access EDITOR"),
            _ => {
                bail!("no editor specified: please use --editor or set EDITOR")
            }
        },
    };

    let out_file = temp_dir.path().join("git-rebase-todo");
    {
        let infile = fs::File::open(&args.file).context("open input file")?;
        let outfile = fs::File::create(&out_file).context("create rebase todo")?;
        restack::restack("origin", infile, outfile)?;
    };

    let exit_code = process::Command::new("sh")
        .arg("-c")
        .arg(format!("{} \"$1\"", editor))
        .arg(editor.as_ref())
        .arg(&out_file)
        .status()
        .context("run editor")?;
    if !exit_code.success() {
        bail!("editor returned non-zero status: {}", exit_code);
    }

    crate::io::rename(&out_file, &args.file).with_context(|| {
        format!(
            "overwrite {} with {}",
            &args.file.display(),
            &out_file.display()
        )
    })
}
