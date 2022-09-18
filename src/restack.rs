//! Implements the core restacking logic.

use crate::git;
use anyhow::{Context, Result};
use std::{
    io::{self, BufRead, Write},
    path,
};

pub struct Config<'a, Git>
where
    Git: git::Git,
{
    git: Git,
    cwd: &'a path::Path,
}

impl<'a, Git> Config<'a, Git>
where
    Git: git::Git,
{
    pub fn new(cwd: &'a path::Path, git: Git) -> Self {
        Self { cwd, git }
    }

    /// restacks branches in the instruction list at `src` and writes the result to
    /// `dst`.
    pub fn restack<I: io::Read, O: io::Write>(
        &self,
        remote_name: &str,
        src: I,
        dst: O,
    ) -> Result<()> {
        let rebase_branch_name = self
            .git
            .rebase_head_name(self.cwd)
            .context("determine the rebase head")?;
        let all_branches = self.git.list_branches(self.cwd).context("list branches")?;

        // TODO: sort all_branches by oid. maybe write a OidMap.

        let src = io::BufReader::new(src);
        let mut restack = Restack {
            remote_name: Some(remote_name),
            rebase_branch_name: &rebase_branch_name,
            last_line_branches: Vec::new(),
            updated_branches: Vec::new(),
            wrote_push: false,
            dst: io::BufWriter::new(dst),
        };

        for line in src.lines() {
            let line = line.context("read input")?;
            if line.is_empty() {
                // Empty lines delineate sections.
                // Write pending "git branch -x" statements
                // before going on to the next section.
                if !restack.update_previous_branches()? {
                    // update_previous_branches adds a trailing newline
                    // only if git branch statements were added.
                    // So if it didn't do anything, re-add the empty line.
                    restack.write_line("")?;
                }
            }

            // Comments usually mark the end of instructions.
            // Flush optional "git push" statements.
            if line.get(0..1) == Some("#") {
                restack
                    .write_push_section(false, true)
                    .context("write remote ref push section")?;
            }

            // (p[ick]|f[ixup]|s[quash]) hash ...
            let mut parts = line.splitn(3, ' ');

            let cmd = parts.next();
            if let Some(cmd) = cmd {
                match cmd {
                    "f" | "fixup" | "s" | "squash" => {} // do nothing
                    _ => {
                        restack.update_previous_branches()?;
                    }
                }
            }

            // Most lines go as-is.
            restack.write_line(&line)?;

            let cmd = match cmd {
                Some(cmd) => cmd,
                None => continue,
            };
            let hash = match cmd {
                "p" | "pick" | "r" | "reword" | "e" | "edit" => match parts.next() {
                    Some(s) => s,
                    None => continue,
                },
                _ => continue,
            };

            restack
                .last_line_branches
                .extend(all_branches.iter().filter(|b| b.shorthash == hash))
        }

        restack.update_previous_branches()?;
        restack.write_push_section(true, false)?;

        Ok(())
    }
}

struct Restack<'a, O: io::Write> {
    remote_name: Option<&'a str>,
    rebase_branch_name: &'a str,
    last_line_branches: Vec<&'a git::Branch>,
    updated_branches: Vec<&'a git::Branch>,
    wrote_push: bool,
    dst: io::BufWriter<O>,
}

impl<'a, O: io::Write> Restack<'a, O> {
    fn write_push_section(&mut self, pad_before: bool, pad_after: bool) -> Result<()> {
        if self.wrote_push {
            return Ok(());
        }
        self.wrote_push = true;

        if self.updated_branches.is_empty() {
            return Ok(());
        }
        let remote_name = match self.remote_name {
            Some(r) => r,
            None => return Ok(()),
        };

        if pad_before {
            writeln!(self.dst)?;
        }
        writeln!(self.dst, "# Uncomment this section to push changes.")?;
        for br in &self.updated_branches {
            writeln!(self.dst, "# exec git push -f {} {}", remote_name, br.name)?;
        }
        if pad_after {
            writeln!(self.dst)?;
        }

        Ok(())
    }

    // Adds "exec git branch -f" statements to the instruction list.
    // Reports whether any statements were added.
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
        writeln!(self.dst, "{}", line).context("write output")
    }
}

// #[cfg(test)]
// mod tests {
//     use super::*;
//     use anyhow::Result;

//     fn restack_test_case(
//         remote_name: &str,
//         rebase_head_name: &str,
//         branches: &[git::Branch],
//         give: &[&str],
//         want: &[&str],
//     ) -> Result<()> {
//         Ok(())
//     }
// }
