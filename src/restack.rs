//! Implements the core restacking logic.

use std::collections::{HashMap, LinkedList};
use std::io::{self, BufRead, Write};
use std::path;

use anyhow::{Context, Result};

use crate::git;

#[cfg(test)]
mod tests;

/// Configures a Restack operation and provides the ability to run it.
pub struct Config<'a, Git>
where
    Git: git::Git,
{
    /// Defines how to talk to Git during the restack.
    git: Git,

    /// Current working directory inside which restack is running.
    cwd: &'a path::Path,
}

impl<'a, Git> Config<'a, Git>
where
    Git: git::Git,
{
    /// Builds a new restack configuration operating inside the given directory
    /// using the provided git object.
    pub fn new(cwd: &'a path::Path, git: Git) -> Self {
        Self { git, cwd }
    }

    /// Reads a git rebase instruction list from `src`
    /// and writes a new instruction list to `dst`
    /// that updates branches found at intermediate commits to their new positions.
    ///
    /// If `remote_name` is specified, includes an opt-in section
    /// at the bottom of the instruction list that
    /// updates the remotes for all affected branches.
    pub fn restack<I: io::Read, O: io::Write>(
        &self,
        remote_name: Option<&str>,
        src: I,
        dst: O,
    ) -> Result<()> {
        let rebase_branch_name = self
            .git
            .rebase_head_name(self.cwd)
            .context("Could not determine rebase head name")?;
        let all_branches = self
            .git
            .list_branches(self.cwd)
            .context("Unable to list branches")?;

        let mut known_branches: HashMap<&str, LinkedList<&git::Branch>> = Default::default();
        all_branches.iter().for_each(|b| {
            known_branches.entry(&b.shorthash).or_default().push_back(b);
        });

        let src = io::BufReader::new(src);
        let mut restack = Restack {
            remote_name,
            rebase_branch_name: &rebase_branch_name,
            dst: io::BufWriter::new(dst),
            known_branches: &known_branches,
            last_line_branches: Vec::new(),
            updated_branches: Vec::new(),
            wrote_push: false,
        };

        for line in src.lines() {
            let line = line.context("Failed while reading input")?;
            restack.process(&line)?;
        }

        restack.update_previous_branches()?;
        restack.write_push_section(true, false)?;

        Ok(())
    }
}

/// Holds state for an ongoing restack operation.
struct Restack<'a, O: io::Write> {
    /// Name of the remote, if any.
    remote_name: Option<&'a str>,

    /// Name of the branch we're rebasing.
    rebase_branch_name: &'a str,

    /// Destination writer.
    dst: io::BufWriter<O>,

    /// Known branches, keyed by their short hashes.
    known_branches: &'a HashMap<&'a str, LinkedList<&'a git::Branch>>,

    last_line_branches: Vec<&'a git::Branch>,
    updated_branches: Vec<&'a git::Branch>,
    wrote_push: bool,
}

impl<'a, O: io::Write> Restack<'a, O> {
    pub fn process(&mut self, line: &str) -> Result<()> {
        if line.is_empty() {
            // Empty lines delineate sections.
            // Write pending "git branch -x" statements
            // before going on to the next section.
            if !self.update_previous_branches()? {
                // update_previous_branches adds a trailing newline
                // only if git branch statements were added.
                // So if it didn't do anything, re-add the empty line.
                self.write_line("")?;
            }
            return Ok(());
        }

        // Comments usually mark the end of instructions.
        // Flush optional "git push" statements.
        if line.get(0..1) == Some("#") {
            self.update_previous_branches()?;
            self.write_push_section(false, true)
                .context("Could not write 'git push' section")?;
        }

        // (p[ick]|f[ixup]|s[quash]) hash ...
        let mut parts = line.splitn(3, ' ');

        let cmd = parts.next();
        if let Some(cmd) = cmd {
            match cmd {
                "f" | "fixup" | "s" | "squash" => {}, // do nothing
                _ => {
                    self.update_previous_branches()?;
                },
            }
        }

        // Most lines go as-is.
        self.write_line(line)?;

        let Some(cmd) = cmd else { return Ok(()); };
        let hash = match cmd {
            "p" | "pick" | "r" | "reword" | "e" | "edit" => match parts.next() {
                Some(s) => s,
                None => return Ok(()),
            },
            _ => return Ok(()),
        };

        if let Some(branches) = self.known_branches.get(hash) {
            self.last_line_branches.extend(branches);
        }

        Ok(())
    }

    fn write_push_section(&mut self, pad_before: bool, pad_after: bool) -> Result<()> {
        if self.wrote_push {
            return Ok(());
        }
        self.wrote_push = true;

        if self.updated_branches.is_empty() {
            return Ok(());
        }

        let Some(remote_name) = self.remote_name else { return Ok(()); };

        if pad_before {
            writeln!(self.dst)?;
        }
        writeln!(self.dst, "# Uncomment this section to push the changes.")?;
        for br in &self.updated_branches {
            writeln!(self.dst, "# exec git push -f {} {}", remote_name, br.name)?;
        }
        if pad_after {
            writeln!(self.dst)?;
        }

        Ok(())
    }

    /// Adds "exec git branch -f" statements to the instruction list.
    /// Reports whether any statements were added.
    fn update_previous_branches(&mut self) -> Result<bool> {
        let mut updated = false;
        for b in self.last_line_branches.drain(0..) {
            if b.name.as_str() == self.rebase_branch_name {
                continue;
            }

            writeln!(self.dst, "exec git branch -f {}", b.name)?;
            self.updated_branches.push(b);
            updated = true;
        }

        if updated {
            writeln!(self.dst)?;
        }

        Ok(updated)
    }

    fn write_line(&mut self, line: &str) -> Result<()> {
        writeln!(self.dst, "{}", line).map_err(Into::into)
    }
}
