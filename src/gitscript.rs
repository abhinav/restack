use std::{env, path};

use anyhow::Result;
use lazy_static::lazy_static;
use restack_testtools::gitscript;

/// Root of the project directory.
///
/// All fixtures and generated archives are in a "fixtures" directory relative
/// to this path.
const CARGO_MANIFEST_DIR: &str = env!("CARGO_MANIFEST_DIR");

type Fixture<'a> = gitscript::Fixture<'a>;

lazy_static! {
    static ref DEFAULT_GROUP: gitscript::Group =
        gitscript::Group::new(path::PathBuf::from(CARGO_MANIFEST_DIR).join("fixtures"));
}

/// Opens and returns a Fixture for the script at `script_path`,
/// ensuring that the archive for the script exists and is up to date.
///
/// The path must be relative to the "fixtures" directory of the repository.
pub fn open<P: AsRef<path::Path>>(script_path: P) -> Result<Fixture<'static>> {
    DEFAULT_GROUP.open(script_path).map_err(anyhow::Error::from)
}
