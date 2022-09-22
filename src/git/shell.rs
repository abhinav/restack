//! Implements the Git trait by shelling out to git.

use std::borrow::Cow;
#[cfg(test)]
use std::collections::HashMap;
use std::fmt::Write;
use std::io::{self, BufRead};
use std::{ffi, path, process};

use anyhow::{bail, Context, Result};

use super::{Branch, Git};

/// Shell provides access to the git CLI.
#[derive(Default)]
pub struct Shell {
    /// envs is only available during tests and provides environment variable
    /// overrides.
    #[cfg(test)]
    envs: HashMap<ffi::OsString, ffi::OsString>,
}

impl Shell {
    /// Builds a new Shell, searching `$PATH` for a git executable.
    pub fn new() -> Self {
        Default::default()
    }

    /// Adds an environment variable to be set for git invocations.
    #[cfg(test)]
    pub fn env<K, V>(&mut self, k: K, v: V) -> &mut Self
    where
        K: AsRef<ffi::OsStr>,
        V: AsRef<ffi::OsStr>,
    {
        self.envs
            .insert(k.as_ref().to_os_string(), v.as_ref().to_os_string());
        self
    }

    /// Builds a `process::Command` for internal use.
    #[allow(clippy::let_and_return, unused_mut)] // used in test
    fn cmd(&self) -> process::Command {
        let mut cmd = process::Command::new("git");
        #[cfg(test)]
        {
            cmd.envs(&self.envs);
        }

        cmd
    }
}

impl Git for Shell {
    fn set_global_config_str<K, V>(&self, k: K, v: V) -> Result<()>
    where
        K: AsRef<ffi::OsStr>,
        V: AsRef<ffi::OsStr>,
    {
        run_cmd(self.cmd().args(&["config", "--global"]).arg(k).arg(v))
    }

    /// git_dir reports the path to the .git directory for the provided directory.
    fn git_dir(&self, dir: &path::Path) -> Result<path::PathBuf> {
        let cmd_out = run_cmd_stdout(
            self.cmd()
                .args(&["rev-parse", "--git-dir"])
                .current_dir(dir),
        )?;

        let mut cmd_out =
            String::from_utf8(cmd_out).context("Output of git rev-parse is not valid UTF-8")?;
        cmd_out.truncate(cmd_out.trim_end().len());

        let mut git_dir = path::PathBuf::from(cmd_out);
        if git_dir.is_relative() {
            git_dir = dir.join(git_dir);
        }

        Ok(git_dir)
    }

    fn list_branches(&self, dir: &path::Path) -> Result<Vec<Branch>> {
        let mut cmd = self.cmd();
        cmd.args(&["show-ref", "--heads", "--abbrev"])
            .current_dir(dir)
            .stderr(process::Stdio::inherit())
            .stdout(process::Stdio::piped());
        let mut child = cmd
            .spawn()
            .with_context(|| format!("Unable to run {}", cmd_desc(&cmd)))?;

        let mut branches: Vec<Branch> = Vec::new();
        {
            let stdout = child.stdout.take().unwrap();
            let rdr = io::BufReader::new(stdout);
            for line in rdr.lines() {
                let line = line.context("Could not read 'git show-ref' output")?;
                let mut parts = line.split(' ');
                || -> Option<()> {
                    let hash = parts.next()?;
                    let refname = parts.next()?;
                    let name = refname.strip_prefix("refs/heads/")?;
                    branches.push(Branch {
                        name: name.to_string(),
                        shorthash: hash.to_string(),
                    });

                    Some(())
                }();
            }
        }

        let status = child
            .wait()
            .with_context(|| format!("Unable to run {}", cmd_desc(&cmd)))?;
        if !status.success() {
            bail!("{} failed: {}", cmd_desc(&cmd), status);
        }

        Ok(branches)
    }
}

/// Runs the given command without capturing its output,
/// and reports a meaningful error if it fails with a non-zero status code.
fn run_cmd(cmd: &mut process::Command) -> Result<()> {
    let status = cmd
        .status()
        .with_context(|| format!("Unable to run {}", cmd_desc(cmd)))?;
    if !status.success() {
        bail!("{} failed: {}", cmd_desc(cmd), status);
    }

    Ok(())
}

/// Runs the given command and captures its output.
/// Reports a meaningful error if the command fails with a non-zero status code,
/// or if reading its output failed.
fn run_cmd_stdout(cmd: &mut process::Command) -> Result<Vec<u8>> {
    let out = cmd
        .output()
        .with_context(|| format!("Unable to run {}", cmd_desc(cmd)))?;

    if !out.status.success() {
        let mut errmsg = format!("{} failed: {}", cmd_desc(cmd), out.status);
        if let Ok(stderr) = std::str::from_utf8(&out.stderr) {
            write!(&mut errmsg, "\nstderr: {}", stderr)?;
        }

        bail!(errmsg);
    }

    Ok(out.stdout)
}

/// Generates a meaningful description of a command.
fn cmd_desc(cmd: &process::Command) -> Cow<str> {
    let prog = cmd.get_program().to_string_lossy();
    let subcmd = cmd
        .get_args()
        .map(ffi::OsStr::to_string_lossy)
        .find(|s| !s.starts_with('-'));
    match subcmd {
        Some(subcmd) => Cow::Owned(format!("{} {}", prog, subcmd)),
        None => prog,
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
            format!("{}", err).contains("not a git repository"),
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

        let mut shell = Shell::new();
        shell.env("HOME", &home);

        shell.set_global_config_str("user.name", "Test User")?;

        let stdout = duct::cmd!("git", "config", "user.name")
            .env("HOME", &home)
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
        branch_names.sort();

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
}
