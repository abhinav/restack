//! Implements the `restack edit` command.

use std::borrow::Cow;
use std::{env, fs, path, process};

use anyhow::{anyhow, bail, Context, Result};

use crate::{git, restack};

const USAGE: &str = "\
USAGE:
    restack edit [OPTIONS] <FILE>

Edits the provided rebase instruction list,
adding commands to move affected branches in the stack during the rebase.

To use this command, set it up as your sequence.editor in Git.
See https://github.com/abhinav/restack#setup for more information.

ARGS:
    <FILE>
            Path to the rebase instruction list

OPTIONS:
    -e, --editor <EDITOR>
            Editor to use for rebase instructions.
            Defaults to $EDITOR.

    -h, --help
            Print help information.
";

/// Arguments for the "restack edit" command.
#[derive(Debug, PartialEq, Eq)]
struct Args {
    /// Editor to use, if any.
    /// Defaults to $EDITOR, and if that's not set, to "vim".
    editor: Option<String>,

    /// Path to the rebase instruction list.
    file: path::PathBuf,
}

/// Runs the `restack edit` command.
pub fn run(mut parser: lexopt::Parser) -> Result<()> {
    let args = {
        let mut editor: Option<String> = None;
        let mut file: Option<path::PathBuf> = None;

        while let Some(arg) = parser.next()? {
            match arg {
                lexopt::Arg::Short('e') | lexopt::Arg::Long("editor") => {
                    let value = parser.value()?;
                    let s = value
                        .to_str()
                        .ok_or_else(|| anyhow!("--editor argument is not a valid string"))?;
                    editor = Some(s.to_string());
                },
                lexopt::Arg::Short('h') | lexopt::Arg::Long("help") => {
                    eprint!("{}", USAGE);
                    return Ok(());
                },
                lexopt::Arg::Value(value) => {
                    file = Some(value.into());
                },
                _ => return Err(arg.unexpected().into()),
            }
        }

        let Some(file) = file else { bail!("Please provide a file name"); };

        Args { editor, file }
    };

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
