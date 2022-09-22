//! Builds directories from shell scripts.
//!
//! With gitscript, you can write a shell script that produces a directory,
//! and have it recorded in the repository for later reuse.
//!
//! Each "fixture" has two pieces:
//!
//! - a shell script
//! - an archive that holds the directory generated by the script
//!
//! The module ensures that these two are in-sync:
//! if the archive doesn't exist or becomes out-of-date,
//! gitscript replaces it with a new one that matches the current script's contents.
//!
//! gitscript specifically aims to address building test Git repositories,
//! so it sets a number of Git-specific environment variables
//! when running the shell scripts.
//!
//! All scripts are run with an already initalized repositories.
//!
//! This approach is heavily inspired by Gitoxide's integration testing system.

use std::borrow::Cow;
use std::collections::HashMap;
use std::fmt::Debug;
use std::result::Result as StdResult;
use std::{env, ffi, fs, io, path};

use anyhow::{bail, Context, Result};
use sha2::Digest;

/// Version of the format used by gitscript in generated archives.
/// This helps ensure that we can change the format later.
const VERSION: &str = "2";

/// Handle to a directory generated by gitscript.
///
/// The directory will be deleted automatically when this is dropped.
pub struct Fixture<'a> {
    tempdir: tempfile::TempDir,
    sha: Vec<u8>,
    version: Cow<'a, str>,
}

impl<'a> Debug for Fixture<'a> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Fixture")
            .field("tempdir", &self.tempdir.path())
            .field("sha", &format!("{:0x?}", &self.sha))
            .field("version", &self.version)
            .finish()
    }
}

impl Fixture<'_> {
    /// Reports the directory generated by gitscript.
    pub fn dir(&self) -> &path::Path {
        self.tempdir.path()
    }

    fn open_archive(path: &path::Path) -> Result<Self> {
        let tempdir = tempfile::tempdir().context("Failed to create temporary directory")?;

        {
            let f = fs::File::open(&path).context("Failed to open archive")?;
            let xz_dec = xz2::read::XzDecoder::new(f);
            tar::Archive::new(xz_dec)
                .unpack(tempdir.path())
                .context("Failed to extract archive")?;
        }

        let sha = fs::read(tempdir.path().join("SHA256")).context("Failed to read script hash")?;
        let version = fs::read_to_string(tempdir.path().join("VERSION"))
            .context("Failed to read fixture version")?;

        // TODO: hash validation here and return error

        Ok(Fixture {
            tempdir,
            sha,
            version: Cow::Owned(version),
        })
    }
}

/// getenv is the default implementation of Group.getenv.
fn getenv(s: &str) -> StdResult<String, env::VarError> {
    env::var(s)
}

/// Group is a group of fixtures rooted in the same directory.
/// This exists mostly for testing -- externally, the DEFAULT_GROUP is used.
pub struct Group {
    dir: path::PathBuf,
    getenv: fn(&str) -> StdResult<String, env::VarError>,
}

impl Group {
    pub fn new<P: AsRef<path::Path>>(p: P) -> Self {
        Self {
            dir: p.as_ref().to_path_buf(),
            getenv,
        }
    }

    /// Reports the absolute path to the given file/folder inside the fixtures/
    /// directory.
    fn fixture_path<P: AsRef<path::Path>>(&self, p: P) -> path::PathBuf {
        self.dir.join(p)
    }

    /// Opens and returns a Fixture for the script at `script_path`,
    /// ensuring that the archive for the script exists and is up to date.
    ///
    /// The path must be relative to the "fixtures" directory of the repository.
    pub fn open<P: AsRef<path::Path>>(&self, script_name: P) -> Result<Fixture> {
        let script_path = self.fixture_path(&script_name);
        let archive_path = script_path.with_extension("tar.xz");

        let script_sha = hash_file(&script_path).context("Failed to hash script")?;

        if archive_path.exists() {
            let fix = Fixture::open_archive(&archive_path).with_context(|| {
                format!("Could not load fixture archive {}", archive_path.display())
            })?;
            if fix.sha == script_sha && fix.version == VERSION {
                return Ok(fix);
            }
            eprintln!("archive {} is outdated", archive_path.display());
        }

        // Fail if the archive wasn't checked in and we're already in CI.
        // GitHub workflows always sets "CI=true".
        if let Ok(ci) = (self.getenv)("CI") {
            if ci.as_str() == "true" {
                eprintln!("cannot generate archive {} in CI", archive_path.display());
                eprintln!("please run the test locally and check the archive in.");
                bail!(
                    "archive {} is outdated or does not exist",
                    archive_path.display()
                );
            }
        }

        // TODO: Move new fixture creation into impl Fixture.

        let tempdir = tempfile::tempdir().context("Failed to create temporary directory")?;
        fs::write(tempdir.path().join("SHA256"), &script_sha)
            .context("Failed to write script hash")?;
        fs::write(tempdir.path().join("VERSION"), VERSION.as_bytes())
            .context("Failed to write fixture version")?;

        let script_path_abs = script_path.canonicalize()?;

        let mut new_path = ffi::OsString::new();
        if let Ok(bin) = self.fixture_path("bin").canonicalize() {
            new_path.push(bin);
        }
        new_path.push(":");
        new_path.push((self.getenv)("PATH")?);

        let env = {
            let mut env: HashMap<_, _> = std::env::vars().collect();
            env.remove("GIT_DIR");

            let mut push = |k: &str, v: &str| {
                env.insert(k.to_string(), v.to_string());
            };

            push("GIT_AUTHOR_DATE", "2000-01-01 00:00:00 +0000");
            push("GIT_COMMITTER_DATE", "2000-01-02 00:00:00 +0000");
            push("PATH", new_path.to_str().unwrap());

            env
        };

        duct::cmd!("git", "init", "-b", "main")
            .dir(tempdir.path())
            .full_env(&env)
            .run()?;

        duct::cmd!("git", "config", "user.name", "author")
            .dir(tempdir.path())
            .full_env(&env)
            .run()?;

        duct::cmd!("git", "config", "user.email", "author@example.com")
            .dir(tempdir.path())
            .full_env(&env)
            .run()?;

        duct::cmd!("bash", "-euo", "pipefail", &script_path_abs)
            .dir(tempdir.path())
            .full_env(&env)
            .run()
            .with_context(|| format!("Could not run script {}", script_path_abs.display()))?;

        {
            let f = fs::File::create(&archive_path)
                .with_context(|| format!("Failed to create archive {}", archive_path.display()))?;

            let out = xz2::write::XzEncoder::new(f, 3);
            let mut ar = tar::Builder::new(out);
            ar.append_dir_all(".", tempdir.path())?;
        }

        Ok(Fixture {
            tempdir,
            sha: script_sha,
            version: Cow::Borrowed(VERSION),
        })
    }
}

/// Hashes the file at the given location.
fn hash_file(p: &path::Path) -> Result<Vec<u8>> {
    let mut f = fs::File::open(&p).context("Failed to open file")?;
    let mut hasher = sha2::Sha256::new();
    io::copy(&mut f, &mut hasher).context("Could not hash file contents")?;
    Ok(hasher.finalize().to_vec())
}

#[cfg(test)]
mod tests {
    use anyhow::Result;
    use tempfile;

    use super::*;

    fn getenv(s: &str) -> StdResult<String, env::VarError> {
        match s {
            "CI" => Err(env::VarError::NotPresent),
            _ => env::var(s),
        }
    }

    #[test]
    fn open_end_to_end() -> Result<()> {
        let tempdir = tempfile::tempdir()?;
        let fixtures = tempdir.path();
        let grp = Group {
            dir: fixtures.to_path_buf(),
            getenv,
        };

        let assert_fixture_contents = |want: &str| -> Result<()> {
            let fix = grp.open("test.sh").context("Could not open fixture")?;
            let got = fs::read_to_string(fix.dir().join("bar"))
                .context("Unable to read generated file")?;
            assert_eq!(want, got, "contents of generated file did not match");

            Ok(())
        };

        // Create a fixture for the first time.
        fs::write(fixtures.join("test.sh"), "echo foo > bar")
            .context("Failed to write test file")?;
        assert_fixture_contents("foo\n").context("Unable to create initial fixture from script")?;

        // Verify archive exists and read from it.
        let archive_hash =
            hash_file(&fixtures.join("test.tar.xz")).context("Failed to hash generated archive")?;
        assert_fixture_contents("foo\n").context("Unable to load fixture from archive")?;

        // Invalidate the archive and reopen.
        fs::write(fixtures.join("test.sh"), "echo 'baz qux' > bar")
            .context("Failed to overwrite test file")?;
        assert_fixture_contents("baz qux\n").context("Unable to overwrite outdated archive")?;
        let new_archive_hash =
            hash_file(&fixtures.join("test.tar.xz")).context("Failed to hash updated archive")?;

        assert_ne!(
            archive_hash, new_archive_hash,
            "generated archive should be updated when fixture changes"
        );

        Ok(())
    }

    #[test]
    fn bad_fixture_script() -> Result<()> {
        let tempdir = tempfile::tempdir()?;
        let fixtures = tempdir.path();
        let grp = Group {
            dir: fixtures.to_path_buf(),
            getenv,
        };

        fs::write(fixtures.join("test.sh"), "false")?;
        let err = grp
            .open("test.sh")
            .expect_err("fixture execution should fail");
        assert!(
            format!("{}", err).contains("Could not run script"),
            "unexpected message: {}",
            err
        );

        Ok(())
    }

    fn getenv_ci(s: &str) -> StdResult<String, env::VarError> {
        match s {
            "CI" => Ok("true".to_string()),
            _ => env::var(s),
        }
    }

    #[test]
    fn new_archives_not_allowed_in_ci() -> Result<()> {
        let tempdir = tempfile::tempdir()?;
        let fixtures = tempdir.path();
        let grp = Group {
            dir: fixtures.to_path_buf(),
            getenv: getenv_ci,
        };

        fs::write(fixtures.join("test.sh"), "echo foo > bar")?;
        let err = grp
            .open("test.sh")
            .expect_err("fixture execution should fail");
        assert!(
            format!("{}", err).contains("outdated or does not exist"),
            "unexpected message: {}",
            err
        );

        Ok(())
    }
}
