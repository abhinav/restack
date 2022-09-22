//! Gates access to Git for the rest of restack.
//! None of the production code in restack should call git directly.

use std::io::{self, Read};
use std::{ffi, fs, path};

use anyhow::{bail, Context, Result};

mod shell;

pub use self::shell::*;

/// A branch in the repository.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Branch {
    /// Name of the branch.
    pub name: String,
    /// Abbreviated hash of the branch commit.
    pub shorthash: String,
}

/// Provides access to Git.
pub trait Git {
    /// Modifies the current user's global git configuration.
    fn set_global_config_str<K, V>(&self, k: K, v: V) -> Result<()>
    where
        K: AsRef<ffi::OsStr>,
        V: AsRef<ffi::OsStr>;

    /// Reports the path to the ".git" directory of the working tree at the
    /// specified path.
    fn git_dir(&self, dir: &path::Path) -> Result<path::PathBuf>;

    /// Returns a list of branches inside the repository at the given path.
    fn list_branches(&self, dir: &path::Path) -> Result<Vec<Branch>>;

    /// Reports the name of the branch currently being rebased at the given path, if any.
    fn rebase_head_name(&self, dir: &path::Path) -> Result<String> {
        let git_dir = self.git_dir(dir).context("Failed to find .git directory")?;
        rebase_head_name(&git_dir)
    }
}

const REBASE_STATE_DIRS: &[&str] = &["rebase-apply", "rebase-merge"];

/// Reports the branch currently being rebased.
///
/// This functionality is not supported natively by the `git` command.
/// The logic was borrowed from `git`'s [own implementation][1].
///
/// [1]: https://github.com/git/git/blob/2f0e14e649d69f9535ad6a086c1b1b2d04436ef5/wt-status.c#L1473
fn rebase_head_name(git_dir: &path::Path) -> Result<String> {
    for state_dir in REBASE_STATE_DIRS {
        let head_file = git_dir.join(state_dir).join("head-name");
        match fs::File::open(&head_file) {
            Err(err) => {
                if err.kind() != io::ErrorKind::NotFound {
                    return Err(err)
                        .with_context(|| format!("Failed to open {}", head_file.display()));
                }
            },
            Ok(mut f) => {
                let mut name = String::new();
                f.read_to_string(&mut name).with_context(|| {
                    format!("Failed to read rebase state from {}", head_file.display())
                })?;

                let name = name.trim();
                return Ok(name.strip_prefix("refs/heads/").unwrap_or(name).to_string());
            },
        }
    }

    // TODO: Use a custom error
    bail!("repository {} is not currently rebasing", git_dir.display())
}

#[cfg(test)]
mod tests {
    use std::os::unix::prelude::PermissionsExt;

    use super::*;
    use crate::git::Shell;
    use crate::gitscript;

    #[test]
    fn not_a_repository() -> Result<()> {
        let tempdir = tempfile::tempdir()?;
        let dir = tempdir.path();

        let git = Shell::new();
        let err = git.rebase_head_name(dir).unwrap_err();
        assert!(
            format!("{}", err).contains("find .git directory"),
            "got error: {}",
            err
        );

        Ok(())
    }

    #[test]
    fn not_currently_rebasing() -> Result<()> {
        let fixture = gitscript::open("empty_commit.sh")?;

        let git = Shell::new();
        let err = git.rebase_head_name(fixture.dir()).unwrap_err();
        assert!(
            format!("{}", err).contains("is not currently rebasing"),
            "got error: {}",
            err
        );

        Ok(())
    }

    #[test]
    fn corrpt_rebase_state_unable_to_open() -> Result<()> {
        let fixture = gitscript::open("empty_commit.sh")?;
        {
            let mut path = fixture.dir().join(".git/rebase-apply");
            fs::create_dir(&path)?;

            path.push("head-name");
            std::fs::write(&path, &[])?;

            let mut perm = fs::metadata(&path)?.permissions();
            perm.set_mode(0o200);
            fs::set_permissions(&path, perm)?
        }

        let git = Shell::new();
        let err = git.rebase_head_name(fixture.dir()).unwrap_err();
        assert!(
            format!("{}", err).contains("Failed to open"),
            "got error: {}",
            err
        );

        Ok(())
    }

    #[test]
    fn corrupt_rebase_state_not_a_file() -> Result<()> {
        let fixture = gitscript::open("empty_commit.sh")?;
        {
            let path = fixture.dir().join(".git/rebase-apply/head-name");
            fs::create_dir_all(&path)?;
        }

        let git = Shell::new();
        let err = git.rebase_head_name(fixture.dir()).unwrap_err();
        assert!(
            format!("{}", err).contains("Failed to read rebase state"),
            "got error: {}",
            err
        );

        Ok(())
    }

    #[test]
    fn mid_rebase() -> Result<()> {
        let fixture = gitscript::open("mid_rebase.sh")?;

        let git_shell = Shell::new();
        let rebase_head = git_shell.rebase_head_name(fixture.dir())?;

        assert_eq!(rebase_head, "feature2");

        Ok(())
    }
}
