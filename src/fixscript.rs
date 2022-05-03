//! Builds directories from shell scripts.
//!
//! With fixscript, you can write a shell script that produces a directory,
//! and have it recorded in the repository for later reuse.
//!
//! Each "fixture" has two pieces:
//!
//! - a shell script
//! - an archive that holds the directory generated by the script
//!
//! The module ensures that these two are in-sync:
//! if the archive doesn't exist or becomes out-of-date,
//! fixscript replaces it with a new one that matches the current script's contents.
//!
//! fixscript specifically aims to address building test Git repositories,
//! so it sets a number of Git-specific environment variables
//! when running the shell scripts.
//!
//! This approach is heavily inspired by Gitoxide's integration testing system.

use anyhow::{bail, Context, Result};
use sha2::Digest;
use std::{env, ffi, fs, io, path, process};

/// Handle to a directory generated by fixscript.
///
/// The directory will be deleted automatically when this is dropped.
pub struct Fixture {
    tempdir: tempfile::TempDir,
}

impl Fixture {
    /// Reports the directory generated by fixscript.
    pub fn dir(&self) -> &path::Path {
        self.tempdir.path()
    }
}

/// Root of the project directory.
///
/// All fixtures and generated archives are in a "fixtures" directory relative
/// to this path.
const CARGO_MANIFEST_DIR: &str = env!("CARGO_MANIFEST_DIR");

/// Reports the absolute path to the given file/folder inside the fixtures/
/// directory.
fn fixture_path<P: AsRef<path::Path>>(p: P) -> path::PathBuf {
    let mut d = path::PathBuf::from(CARGO_MANIFEST_DIR);
    d.push("fixtures");
    d.join(p)
}

/// Hashes the file at the given location.
fn hash_file(p: &path::Path) -> Result<Vec<u8>> {
    let mut f = fs::File::open(&p).context("open file")?;
    let mut hasher = sha2::Sha256::new();
    io::copy(&mut f, &mut hasher).context("hash file contents")?;
    Ok(hasher.finalize().to_vec())
}

/// Version of the format used by fixscript in generated archives.
/// This helps ensure that we can change the format later.
const VERSION: &str = "1";

/// Opens and returns a Fixture for the script at `script_path`,
/// ensuring that the archive for the script exists and is up to date.
///
/// The path must be relative to the "fixtures" directory of the repository.
pub fn open<P: AsRef<path::Path>>(script_path: P) -> Result<Fixture> {
    let script_path = fixture_path(script_path);
    let archive_path = script_path.with_extension("tar.xz");

    let script_sha = hash_file(&script_path).context("hash script")?;

    if archive_path.exists() {
        let tempdir = tempfile::tempdir().context("create temporary directory")?;

        {
            let f = fs::File::open(&archive_path)
                .with_context(|| format!("open {}", archive_path.display()))?;
            let xz_dec = xz2::read::XzDecoder::new(f);
            tar::Archive::new(xz_dec)
                .unpack(tempdir.path())
                .with_context(|| format!("extract {}", archive_path.display()))?;
        }

        let got_sha = fs::read(tempdir.path().join("SHA256")).context("read SHA file")?;
        let got_version =
            fs::read_to_string(tempdir.path().join("VERSION")).context("read VERSION file")?;

        if got_sha == script_sha && got_version == VERSION {
            return Ok(Fixture { tempdir });
        }

        eprintln!("archive {} is outdated", archive_path.display());
    }

    // Fail if the archive wasn't checked in and we're already in CI.
    // GitHub workflows always sets "CI=true".
    if let Ok(ci) = env::var("CI") {
        if ci.as_str() == "true" {
            eprintln!("cannot generate archive {} in CI", archive_path.display());
            eprintln!("please run the test locally and check the archive in.");
            bail!(
                "archive {} is outdated or does not exist",
                archive_path.display()
            );
        }
    }

    let tempdir = tempfile::tempdir().context("create temporary directory")?;
    fs::write(tempdir.path().join("SHA256"), &script_sha).context("write SHA file")?;
    fs::write(tempdir.path().join("VERSION"), VERSION.as_bytes()).context("write VERSION file")?;

    let script_path_abs = script_path.canonicalize()?;

    let mut new_path = ffi::OsString::new();
    new_path.push(fixture_path("bin").canonicalize()?);
    new_path.push(":");
    new_path.push(env::var("PATH")?);

    let status = process::Command::new("bash")
        .args(&["-euo", "pipefail"]) // disallow failures
        .arg(script_path_abs)
        .current_dir(tempdir.path())
        .env_remove("GIT_DIR")
        .env("GIT_AUTHOR_DATE", "2000-01-01 00:00:00 +0000")
        .env("GIT_AUTHOR_EMAIL", "author@example.com")
        .env("GIT_AUTHOR_NAME", "author")
        .env("GIT_COMMITTER_DATE", "2000-01-02 00:00:00 +0000")
        .env("GIT_COMMITTER_EMAIL", "committer@example.com")
        .env("GIT_COMMITTER_NAME", "committer")
        .env("PATH", &new_path)
        .status()?;
    if !status.success() {
        bail!("fixture {} failed", script_path.display());
    }

    {
        let f = fs::File::create(&archive_path)
            .with_context(|| format!("create {}", archive_path.display()))?;

        let out = xz2::write::XzEncoder::new(f, 3);
        let mut ar = tar::Builder::new(out);
        ar.append_dir_all(".", tempdir.path())?;
    }

    Ok(Fixture { tempdir })
}
