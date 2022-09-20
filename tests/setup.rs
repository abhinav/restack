use anyhow::{Context, Result};
use std::{env, fs, os::unix::fs::PermissionsExt};
use tempfile::tempdir;

const RESTACK: &str = env!("CARGO_BIN_EXE_restack");

#[test]
fn prints_edit_script() -> Result<()> {
    let stdout = duct::cmd!(RESTACK, "setup", "--print-edit-script").read()?;
    let first_line = stdout.lines().next().expect("non empty output");
    assert_eq!(first_line, "#!/bin/sh -e");

    Ok(())
}

#[test]
fn setup_restack() -> Result<()> {
    let home_dir = tempdir().context("Failed to make temporary directory")?;

    duct::cmd!(RESTACK, "setup")
        .env("HOME", home_dir.path())
        .run()?;

    let edit_script = home_dir.path().join(".restack/edit.sh");
    assert!(edit_script.exists(), "edit script does not exist");
    {
        let mode = edit_script.metadata()?.permissions().mode();
        assert_ne!(mode & 0o111, 0, "file should be executable, got {}", mode);
    }

    let stdout = duct::cmd!("git", "config", "--global", "sequence.editor")
        .env("HOME", home_dir.path())
        .read()?;
    assert_eq!(edit_script.to_str().unwrap(), stdout.trim_end());

    Ok(())
}

#[test]
fn update_old_setup() -> Result<()> {
    let home_dir = tempdir().context("Failed to make temporary directory")?;
    let edit_script = home_dir.path().join(".restack/edit.sh");

    // Outdated setup:
    fs::create_dir(edit_script.parent().unwrap())?;
    fs::write(&edit_script, "old script".as_bytes())?;
    duct::cmd!("git", "config", "--global", "sequence.editor", "nvim")
        .env("HOME", home_dir.path())
        .run()?;

    // Overwrite it.
    duct::cmd!(RESTACK, "setup")
        .env("HOME", home_dir.path())
        .run()?;
    let stdout = duct::cmd!("git", "config", "--global", "sequence.editor")
        .env("HOME", home_dir.path())
        .read()?;
    assert_eq!(edit_script.to_str().unwrap(), stdout.trim_end());

    Ok(())
}
