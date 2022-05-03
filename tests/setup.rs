use anyhow::{Context, Result};
use std::{
    env, fs,
    io::BufRead,
    process::{Command, Stdio},
};
use tempfile::tempdir;

const RESTACK: &str = env!("CARGO_BIN_EXE_restack");

#[test]
fn prints_edit_script() -> Result<()> {
    let out = Command::new(RESTACK)
        .args(&["setup", "--print-edit-script"])
        .stderr(Stdio::inherit())
        .output()
        .context("run restack")?;

    assert!(out.status.success(), "restack failed");
    let first_line = out
        .stdout
        .lines()
        .next()
        .expect("non empty output")
        .context("read stdout")?;
    assert_eq!(first_line, "#!/bin/sh -e");

    Ok(())
}

#[test]
fn setup_restack() -> Result<()> {
    let home_dir = tempdir().context("make tempdir")?;

    {
        let status = Command::new(RESTACK)
            .arg("setup")
            .env("HOME", home_dir.path())
            .status()
            .context("run restack")?;
        assert!(status.success(), "restack failed");
    }

    let edit_script = home_dir.path().join(".restack/edit.sh");
    assert!(edit_script.exists(), "edit script does not exist");

    {
        let out = Command::new("git")
            .args(&["config", "--global", "sequence.editor"])
            .env("HOME", home_dir.path())
            .stderr(Stdio::inherit())
            .output()
            .context("run git")?;
        assert!(out.status.success(), "git failed");

        assert_eq!(
            edit_script.to_str().unwrap(),
            std::str::from_utf8(&out.stdout).unwrap().trim_end(),
        );
    }

    Ok(())
}

#[test]
fn update_old_setup() -> Result<()> {
    let home_dir = tempdir().context("make tempdir")?;
    let edit_script = home_dir.path().join(".restack/edit.sh");

    // Outdated setup:
    {
        fs::create_dir(edit_script.parent().unwrap())?;
        fs::write(&edit_script, "old script".as_bytes())?;

        let status = Command::new("git")
            .args(&["config", "--global", "sequence.editor", "nvim"])
            .env("HOME", home_dir.path())
            .status()?;
        assert!(status.success(),);
    }

    {
        let status = Command::new(RESTACK)
            .arg("setup")
            .env("HOME", home_dir.path())
            .status()
            .context("run restack")?;
        assert!(status.success(), "restack failed");
    }

    {
        let out = Command::new("git")
            .args(&["config", "--global", "sequence.editor"])
            .env("HOME", home_dir.path())
            .stderr(Stdio::inherit())
            .output()
            .context("run git")?;
        assert!(out.status.success(), "git failed");

        assert_eq!(
            edit_script.to_str().unwrap(),
            std::str::from_utf8(&out.stdout).unwrap().trim_end(),
        );
    }

    Ok(())
}
