//! Gates access to Git for the rest of restack.
//! None of the production code in restack should call git directly.

use anyhow::Result;
use std::{ffi, path};

mod rebase_head;
mod shell;

pub use self::rebase_head::*;
pub use self::shell::*;

/// A branch in the repository.
#[derive(Debug, PartialEq, Eq)]
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
}
