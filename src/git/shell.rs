//! Implements the Git trait by shelling out to git.

use std::io::{self, BufRead};
use std::{ffi, path, process};

use anyhow::{Context, Result};

use super::{Branch, Git};
use crate::exec::ExitStatusExt;

/// Shell provides access to the git CLI.
pub struct Shell {}

impl Shell {
    /// Builds a new Shell, searching `$PATH` for a git executable.
    pub fn new() -> Self {
        Self {}
    }

    /// Builds a `process::Command` for internal use.
    fn cmd(&self) -> process::Command {
        let mut cmd = process::Command::new("git");
        cmd.stderr(process::Stdio::inherit());

        cmd
    }
}

impl Git for Shell {
    fn set_global_config_str<K, V>(&self, k: K, v: V) -> Result<()>
    where
        K: AsRef<ffi::OsStr>,
        V: AsRef<ffi::OsStr>,
    {
        self.cmd()
            .args(["config", "--global"])
            .arg(k)
            .arg(v)
            .status()
            .context("Unable to run git config")?
            .exit_ok()
            .context("git config failed")
    }

    /// `git_dir` reports the path to the .git directory for the provided directory.
    fn git_dir(&self, dir: &path::Path) -> Result<path::PathBuf> {
        let out = self
            .cmd()
            .args(["rev-parse", "--git-dir"])
            .current_dir(dir)
            .output()
            .context("Failed to run git rev-parse")?;

        out.status.exit_ok().context("git rev-parse failed")?;

        let output = std::str::from_utf8(out.stdout.trim_ascii_end())
            .context("Output of git rev-parse is not valid UTF-8")?;

        let mut git_dir = path::PathBuf::from(output);
        if git_dir.is_relative() {
            git_dir = dir.join(git_dir);
        }

        Ok(git_dir)
    }

    fn list_branches(&self, dir: &path::Path) -> Result<Vec<Branch>> {
        let mut cmd = self.cmd();
        cmd.args(["show-ref", "--heads", "--abbrev"])
            .current_dir(dir)
            .stdout(process::Stdio::piped());
        let mut child = cmd.spawn().context("Unable to run git show-ref")?;

        let mut branches: Vec<Branch> = Vec::new();
        let Some(stdout) = child.stdout.take() else {
            unreachable!("Stdio::piped() always sets child.stdout");
        };
        {
            let rdr = io::BufReader::new(stdout);
            for line in rdr.lines() {
                let line = line.context("Could not read 'git show-ref' output")?;
                let mut parts = line.split(' ');
                // Output of git show-ref is in the form,
                //   $hash1 refs/heads/$name1
                //   $hash2 refs/heads/$name2

                let Some(hash) = parts.next() else { continue };
                let Some(refname) = parts.next() else {
                    continue;
                };
                let Some(name) = refname.strip_prefix("refs/heads/") else {
                    continue;
                };

                branches.push(Branch {
                    name: name.to_string(),
                    shorthash: hash.to_string(),
                });
            }
        }

        child
            .wait()
            .context("Unable to start git show-ref")?
            .exit_ok()
            .context("git show-ref failed")?;

        Ok(branches)
    }

    fn comment_string(&self, dir: &path::Path) -> Result<String> {
        let out = self
            .cmd()
            // Looking up git config for a field that is unset
            // will return a non-zero exit code
            // if we don't specify a default value.
            .args(["config", "--get", "core.commentString"])
            .current_dir(dir)
            .output()
            .context("Failed to run git config")?;
        let output = match out.status.code() {
            Some(0) => std::str::from_utf8(out.stdout.trim_ascii_end())
                .context("Output of git config is not valid UTF-8")?
                .to_string(),

            _ => {
                // Fall back to core.commentChar if core.commentString is unset.
                let out = self
                    .cmd()
                    // Looking up git config for a field that is unset
                    // will return a non-zero exit code
                    // if we don't specify a default value.
                    .args(["config", "--get", "--default=#", "core.commentChar"])
                    .current_dir(dir)
                    .output()
                    .context("Failed to run git config")?;
                out.status.exit_ok().context("git config failed")?;

                std::str::from_utf8(out.stdout.trim_ascii_end())
                    .context("Output of git config is not valid UTF-8")?
                    .to_string()
            },
        };

        match output.as_str() {
            // In auto, git will pick an unused character from a pre-defined list.
            // This might be useful to support in the future.
            "auto" => anyhow::bail!(
                "core.commentChar=auto is not supported yet. \
                 Please set core.commentChar to a single character \
                 or disable restack by unsetting sequence.editor."
            ),

            // Unreachable but easy enough to handle.
            "" => Ok("#".to_string()),

            _ => Ok(output),
        }
    }
}

#[cfg(test)]
mod tests {
    use pretty_assertions::assert_eq;

    use super::*;
    use crate::gitscript;

    #[test]
    fn git_dir() -> Result<()> {
        let fixture = gitscript::open("empty_commit.sh")?;

        let shell = Shell::new();
        let git_dir = shell.git_dir(fixture.dir())?;

        assert_eq!(git_dir, fixture.dir().join(".git"));
        Ok(())
    }

    #[test]
    fn git_dir_not_a_repository() -> Result<()> {
        let tempdir = tempfile::tempdir()?;
        let dir = tempdir.path();

        let shell = Shell::new();
        let err = shell.git_dir(dir).unwrap_err();

        assert!(
            format!("{}", err).contains("rev-parse failed"),
            "got error: {}",
            err
        );

        Ok(())
    }

    #[test]
    fn set_global_config_str() -> Result<()> {
        let workdir = tempfile::tempdir()?;
        let homedir = tempfile::tempdir()?;
        let home = homedir.path();

        // This is hacky but it's the only way to prevent
        // any other environment variables from leaking
        // into the Git subprocess
        unsafe {
            std::env::vars().for_each(|(k, _)| {
                std::env::remove_var(k);
            });
            std::env::set_var("HOME", home);
        }

        let shell = Shell::new();
        shell.set_global_config_str("user.name", "Test User")?;

        let stdout = duct::cmd!("git", "config", "user.name")
            .dir(workdir.path())
            .read()?;

        assert_eq!(stdout.trim(), "Test User");

        Ok(())
    }

    #[test]
    fn list_branches_empty_repo() -> Result<()> {
        let fixture = gitscript::open("empty.sh")?;

        let shell = Shell::new();
        let res = shell.list_branches(fixture.dir());

        assert!(res.is_err(), "expected error, got {:?}", res.unwrap());

        Ok(())
    }

    #[test]
    fn list_branches_single() -> Result<()> {
        let fixture = gitscript::open("empty_commit.sh")?;

        let shell = Shell::new();
        let branches = shell.list_branches(fixture.dir())?;
        assert!(
            branches.len() == 1,
            "expected a single item, got {:?}",
            branches
        );

        let branch = &branches[0];
        assert_eq!(branch.name, "main");
        assert!(!branch.shorthash.is_empty(), "hash should not be empty");

        Ok(())
    }

    #[test]
    fn list_branches_many() -> Result<()> {
        let fixture = gitscript::open("simple_many_branches.sh")?;

        let shell = Shell::new();
        let branches = shell.list_branches(fixture.dir())?;

        let mut branch_names = branches
            .iter()
            .map(|b| b.name.as_ref())
            .collect::<Vec<&str>>();
        branch_names.sort_unstable();

        assert_eq!(
            &["bar", "baz", "foo", "main", "quux", "qux"],
            branch_names.as_slice()
        );

        Ok(())
    }

    #[test]
    fn list_branches_not_a_repository() -> Result<()> {
        let tempdir = tempfile::tempdir()?;
        let dir = tempdir.path();

        let shell = Shell::new();
        let err = shell.list_branches(dir).unwrap_err();

        assert!(
            format!("{}", err).contains("git show-ref failed"),
            "got error: {}",
            err
        );

        Ok(())
    }

    #[test]
    fn comment_char_default() -> Result<()> {
        let fixture = gitscript::open("simple_stack.sh")?;

        let shell = Shell::new();
        let comment_char = shell.comment_string(fixture.dir())?;
        assert!(
            comment_char.as_str() == "#",
            "unexpected comment char: '{}'",
            comment_char
        );

        Ok(())
    }

    #[test]
    fn comment_char() -> Result<()> {
        let fixture = gitscript::open("simple_stack_comment_char.sh")?;

        let shell = Shell::new();
        let comment_char = shell.comment_string(fixture.dir())?;
        assert!(
            comment_char.as_str() == ";",
            "unexpected comment char: '{}'",
            comment_char
        );

        Ok(())
    }

    #[test]
    fn comment_string() -> Result<()> {
        let fixture = gitscript::open("simple_stack_comment_string.sh")?;

        let shell = Shell::new();
        let comment_str = shell.comment_string(fixture.dir())?;
        assert!(
            comment_str == "#:",
            "unexpected comment string: '{}'",
            &comment_str,
        );

        Ok(())
    }

    #[test]
    fn comment_char_auto() -> Result<()> {
        let fixture = gitscript::open("empty_commit_comment_char_auto.sh")?;

        let shell = Shell::new();

        // Should return an error.
        let err = match shell.comment_string(fixture.dir()) {
            Ok(v) => panic!("expected an error, got {:?}", v),
            Err(err) => err,
        };

        assert!(
            format!("{}", err).contains("core.commentChar=auto"),
            "got error: {}",
            err
        );

        Ok(())
    }
}
