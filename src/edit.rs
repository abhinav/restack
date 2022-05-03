//! Implements the `restack edit` command.

use std::{borrow::Cow, env, fs, io, path, process};

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

    match fs::rename(&out_file, &args.file) {
        Ok(_) => Ok(()),
        Err(err) => {
            // If /tmp is mounted to a different partition (it often is),
            // attempting to move the file will cause the error:
            //   invalid cross-device link
            //
            // For that case, fall back to copying the file and
            // deleting the temporary file.
            //
            // This is not the default because move is atomic.
            if err.raw_os_error() == Some(18) {
                // TODO: Use io::ErrorKind::CrossesDevices after
                // https://github.com/rust-lang/rust/issues/86442.

                unsafe_rename(&out_file, &args.file)
            } else {
                Err(anyhow::Error::new(err))
            }
        }
    }
    .with_context(|| {
        format!(
            "overwrite {} with {}",
            &args.file.display(),
            &out_file.display()
        )
    })
}

/// Renames a file by copying its contents into a new file non-atomically,
/// and deleting the original file.
///
/// This is necessary because on Linux, we cannot move the file across
/// filesystem boundaries, and /tmp is often mounted on a different file system
/// than the user's working directory.
fn unsafe_rename(src: &path::Path, dst: &path::Path) -> Result<()> {
    let md = fs::metadata(src).with_context(|| format!("inspect {}", src.display()))?;

    {
        let mut r = fs::File::open(src).context("open source")?;
        let mut w = fs::File::create(dst).context("open destination")?;
        io::copy(&mut r, &mut w).context("copy contents")?;
    }

    fs::set_permissions(dst, md.permissions()).context("update destination permissions")?;
    fs::remove_file(src).context("delete source file")?;

    Ok(())
}
