//! Reports the branch currently being rebased.
//!
//! This functionality is not supported natively by the `git` command.
//! The logic was borrowed from `git`'s [own implementation][1].
//!
//! [1]: https://github.com/git/git/blob/2f0e14e649d69f9535ad6a086c1b1b2d04436ef5/wt-status.c#L1473

use anyhow::{bail, Context, Result};
use std::{
    fs,
    io::{self, Read},
    path,
};

use super::Git;

const REBASE_STATE_DIRS: &[&str] = &["rebase-apply", "rebase-merge"];

/// Reports the name of the branch currently being rebased at the given path, if
/// any.
pub fn rebase_head_name<G: Git>(git: &G, dir: &path::Path) -> Result<String> {
    let git_dir = git.git_dir(dir).context("find .git directory")?;

    for state_dir in REBASE_STATE_DIRS {
        let head_file = git_dir.join(state_dir).join("head-name");
        match fs::File::open(&head_file) {
            Err(err) => {
                if err.kind() != io::ErrorKind::NotFound {
                    return Err(err).with_context(|| format!("inspect {}", head_file.display()));
                }
            }
            Ok(mut f) => {
                let mut name = String::new();
                f.read_to_string(&mut name)
                    .with_context(|| format!("read rebase state from {}", head_file.display()))?;

                let name = name.trim();
                return Ok(name.strip_prefix("refs/heads/").unwrap_or(name).to_string());
            }
        }
    }

    // TODO: Use a custom error
    bail!("repository {} is not currently rebasing", git_dir.display())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{fixscript, git::Shell};

    #[test]
    fn not_a_repository() -> Result<()> {
        let tempdir = tempfile::tempdir()?;
        let dir = tempdir.path();

        let git = Shell::new();
        let err = rebase_head_name(&git, dir).unwrap_err();
        assert!(
            format!("{}", err).contains("find .git directory"),
            "got error: {}",
            err
        );

        Ok(())
    }

    #[test]
    fn not_currently_rebasing() -> Result<()> {
        let fixture = fixscript::open("empty_repo_single_commit.sh")?;

        let git = Shell::new();
        let err = rebase_head_name(&git, fixture.dir()).unwrap_err();
        assert!(
            format!("{}", err).contains("is not currently rebasing"),
            "got error: {}",
            err
        );

        Ok(())
    }

    #[test]
    fn mid_rebase() -> Result<()> {
        let fixture = fixscript::open("mid_rebase.sh")?;

        let git_shell = Shell::new();
        let rebase_head = rebase_head_name(&git_shell, fixture.dir())?;

        assert_eq!(rebase_head, "feature2");

        Ok(())
    }
}
