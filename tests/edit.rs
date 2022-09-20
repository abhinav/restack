use anyhow::{Context, Result};
use lazy_static::lazy_static;
use pretty_assertions::assert_eq;
use restack_testtools::gitscript;
use std::{fs, path};

const RESTACK: &str = env!("CARGO_BIN_EXE_restack");

lazy_static! {
    static ref FIXTURES_DIR: path::PathBuf =
        path::PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("fixtures");
    static ref DEFAULT_GITSCRIPT_GROUP: gitscript::Group =
        gitscript::Group::new(FIXTURES_DIR.as_path());
}

pub fn open_fixture<P>(script_path: P) -> Result<gitscript::Fixture<'static>>
where
    P: AsRef<path::Path>,
{
    DEFAULT_GITSCRIPT_GROUP.open(script_path)
}

#[test]
fn simple_stack() -> Result<()> {
    let repo_fixture = open_fixture("simple_stack.sh")?;

    duct::cmd!(
        "git",
        "config",
        "sequence.editor",
        format!("{} edit", RESTACK)
    )
    .dir(repo_fixture.dir())
    .run()?;

    duct::cmd!("git", "rebase", "--interactive", "main")
        .env("EDITOR", FIXTURES_DIR.join("bin/add_break.sh"))
        .dir(repo_fixture.dir())
        .run()?;

    // add_break.sh should have seen a rebase list
    // with instructions to update branches.
    // To verify this, introduce a new commit at the top of the stack,
    // and verify that that file is present in all branches after the rebase finishes.

    fs::write(repo_fixture.dir().join("README"), "wait for me")?;
    duct::cmd!(
        "bash",
        "-c",
        "git add README &&
        git commit -m 'add README' &&
        git rebase --continue"
    )
    .dir(repo_fixture.dir())
    .run()?;

    let branches = &["foo", "bar", "baz"];
    for br in branches {
        let got = duct::cmd!("git", "show", format!("{}:README", br))
            .dir(repo_fixture.dir())
            .read()
            .with_context(|| format!("Unable to print {}:README", br))?;

        assert_eq!(
            "wait for me", &got,
            "Contents of {}:README do not match",
            br
        );
    }

    Ok(())
}
