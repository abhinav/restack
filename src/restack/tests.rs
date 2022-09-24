#![cfg(test)]

use std::path;

use anyhow::Result;
use indoc::indoc;
use pretty_assertions::assert_eq;

use super::Config;
use crate::git;

struct StubGit {
    branches: Vec<git::Branch>,
    rebase_head_name: String,
}

impl git::Git for StubGit {
    fn set_global_config_str<K, V>(&self, _: K, _: V) -> Result<()>
    where
        K: AsRef<std::ffi::OsStr>,
        V: AsRef<std::ffi::OsStr>,
    {
        unreachable!("set_global_config_str not expected")
    }

    fn git_dir(&self, _: &path::Path) -> Result<path::PathBuf> {
        unreachable!("git_dir not expected")
    }

    fn list_branches(&self, _: &path::Path) -> Result<Vec<git::Branch>> {
        Ok(self.branches.clone())
    }

    fn rebase_head_name(&self, _: &path::Path) -> Result<String> {
        Ok(self.rebase_head_name.clone())
    }
}

struct TestBranch<'a> {
    name: &'a str,
    hash: &'a str,
}

struct TestCase<'a> {
    remote_name: Option<&'a str>,
    rebase_head_name: &'a str,
    branches: &'a [TestBranch<'a>],
    give: &'a str,
    want: &'a str,
}

fn restack_test(test: &TestCase) -> Result<()> {
    let tempdir = tempfile::tempdir()?;
    let branches = test
        .branches
        .iter()
        .map(|b| git::Branch {
            name: b.name.to_owned(),
            shorthash: b.hash.to_owned(),
        })
        .collect();
    let stub_git = StubGit {
        branches,
        rebase_head_name: test.rebase_head_name.to_owned(),
    };

    let cfg = Config::new(tempdir.path(), stub_git);

    let got = {
        let mut got: Vec<u8> = Vec::new();
        cfg.restack(test.remote_name, test.give.as_bytes(), &mut got)?;
        String::from_utf8(got)?
    };

    assert_eq!(test.want, &got);

    Ok(())
}

macro_rules! testcase {
    ($name:ident, $test: expr) => {
        #[test]
        fn $name() -> Result<()> {
            let test = $test;
            restack_test(&test)
        }
    };
}

fn branch<'a>(name: &'a str, hash: &'a str) -> TestBranch<'a> {
    TestBranch { name, hash }
}

testcase!(
    no_matches,
    TestCase {
        remote_name: Some("origin"),
        rebase_head_name: "feature",
        branches: &[branch("feature", "hash1")],
        give: indoc! {"
            pick hash0 Do something
            exec make test
            pick hash1 Implement feature
        "},
        want: indoc! {"
            pick hash0 Do something
            exec make test
            pick hash1 Implement feature
        "},
    }
);

testcase!(
    bad_pick_instruction,
    TestCase {
        remote_name: None,
        rebase_head_name: "foo",
        branches: &[],
        give: indoc! {"
            pick
        "},
        want: indoc! {"
            pick
        "},
    }
);

testcase!(
    branch_at_rebase_head,
    TestCase {
        remote_name: Some("origin"),
        rebase_head_name: "feature/wip",
        branches: &[branch("feature/1", "hash1"), branch("feature/wip", "hash1"),],
        give: indoc! {"
            pick hash0 Do something
            exec make test
            pick hash1 Implement feature
            exec make test

            # Rebase instructions
        "},
        want: indoc! {"
            pick hash0 Do something
            exec make test
            pick hash1 Implement feature
            exec git branch -f feature/1

            exec make test

            # Uncomment this section to push the changes.
            # exec git push -f origin feature/1

            # Rebase instructions
        "},
    }
);

testcase!(
    rebase_instructions_comment_missing,
    TestCase {
        remote_name: Some("origin"),
        rebase_head_name: "feature/wip",
        branches: &[
            branch("feature/1", "hash1"),
            branch("feature/2", "hash3"),
            branch("feature/3", "hash7"),
            branch("feature/wip", "hash9"),
        ],
        give: indoc! {"
            pick hash0 Do something 0
            pick hash1 Implement feature1
            pick hash2 Do something
            pick hash3 Implement feature2
            pick hash4 Do something 4
            pick hash5 Do something 5
            pick hash6 Do something 6
            pick hash7 Implement feature3
            pick hash8 Do something 8
            pick hash9 Do something 9
        "},
        want: indoc! {"
            pick hash0 Do something 0
            pick hash1 Implement feature1
            exec git branch -f feature/1

            pick hash2 Do something
            pick hash3 Implement feature2
            exec git branch -f feature/2

            pick hash4 Do something 4
            pick hash5 Do something 5
            pick hash6 Do something 6
            pick hash7 Implement feature3
            exec git branch -f feature/3

            pick hash8 Do something 8
            pick hash9 Do something 9

            # Uncomment this section to push the changes.
            # exec git push -f origin feature/1
            # exec git push -f origin feature/2
            # exec git push -f origin feature/3
        "},
    }
);

testcase!(
    fixup_commit,
    TestCase {
        remote_name: None,
        rebase_head_name: "b",
        branches: &[branch("a", "hash1"), branch("b", "hash3")],
        give: indoc! {"
            pick hash0 do thing
            pick hash1 another thing
            fixup hash2 stuff
            pick hash3 whatever
        "},
        want: indoc! {"
            pick hash0 do thing
            pick hash1 another thing
            fixup hash2 stuff
            exec git branch -f a

            pick hash3 whatever
        "},
    }
);

testcase!(
    squash_commit,
    TestCase {
        remote_name: None,
        rebase_head_name: "b",
        branches: &[branch("a", "hash1"), branch("b", "hash3")],
        give: indoc! {"
            pick hash0 do thing
            pick hash1 another thing
            squash hash2 stuff
            pick hash3 whatever
        "},
        want: indoc! {"
            pick hash0 do thing
            pick hash1 another thing
            squash hash2 stuff
            exec git branch -f a

            pick hash3 whatever
        "},
    }
);

testcase!(
    issue_41, // https://github.com/abhinav/restack/issues/41
    TestCase {
        remote_name: Some("origin"),
        rebase_head_name: "stack",
        branches: &[
            branch("5601-connection-refused", "29b83a30c"),
            branch("5460-publish-bundle", "ae23c4203"),
            branch("5460-publish-bundle-client", "ea9b3946b"),
        ],
        give: indoc! {"
            pick eaed5a16a <Kris Kowal> fix(agoric-cli): Thread rpcAddresses for Cosmos publishBundle
            pick bd49c28ed <Kris Kowal> fix(agoric-cli): Follow-up: thread random as power
            pick da0626a7e <Kris Kowal> fix(agoric-cli): Follow-up: conditionally coerce RPC addresses
            pick 29b83a30c <Kris Kowal> fix(agoric-cli): Follow-up: heuristic for distinguishing bare hostnames from URLs (5601-connection-refused)
            pick e2b9551ba <Kris Kowal> fix(cosmic-swingset): Publish installation success and failure topic
            pick d8685a017 <Kris Kowal> fix(cosmic-swingset): Follow-up: use new pubsub mechanism
            pick a8ca6046f <Kris Kowal> fix(cosmic-swingset): Follow-up comment from Richard Gibson
            pick 33445b5b7 <Kris Kowal> fix(cosmic-swingset): Follow-up: Publish base value to installation topic
            pick 021edf49e <Kris Kowal> fix(cosmic-swingset): Follow-up: prettier
            pick d0af526c4 <Kris Kowal> fix(cosmic-swingset): Follow-up: should publish sequences
            pick ae23c4203 <Kris Kowal> refactor: Thread solo home directory more generally (5460-publish-bundle)
            pick c8090b4ab <Kris Kowal> refactor(agoric-cli): Publish with RPC instead of agd subshell
            pick 8cd36f76e <Kris Kowal> fix(casting): iterateLatest erroneously adapted getEachIterable
            pick fe113b6bc <Kris Kowal> fix(casting): Release all I/O handles between yield and next
            pick 00d84a3e1 <Kris Kowal> feat(agoric-cli): Reveal block heights to agoric follow, opt-in for lossy
            pick 2c04f46aa <Kris Kowal> chore: Update yarn.lock
            pick ea9b3946b <Kris Kowal> chore: yarn deduplicate (HEAD -> stack, 5460-publish-bundle-client)

            # Rebase comment starts here
        "},
        want: indoc! {"
            pick eaed5a16a <Kris Kowal> fix(agoric-cli): Thread rpcAddresses for Cosmos publishBundle
            pick bd49c28ed <Kris Kowal> fix(agoric-cli): Follow-up: thread random as power
            pick da0626a7e <Kris Kowal> fix(agoric-cli): Follow-up: conditionally coerce RPC addresses
            pick 29b83a30c <Kris Kowal> fix(agoric-cli): Follow-up: heuristic for distinguishing bare hostnames from URLs (5601-connection-refused)
            exec git branch -f 5601-connection-refused

            pick e2b9551ba <Kris Kowal> fix(cosmic-swingset): Publish installation success and failure topic
            pick d8685a017 <Kris Kowal> fix(cosmic-swingset): Follow-up: use new pubsub mechanism
            pick a8ca6046f <Kris Kowal> fix(cosmic-swingset): Follow-up comment from Richard Gibson
            pick 33445b5b7 <Kris Kowal> fix(cosmic-swingset): Follow-up: Publish base value to installation topic
            pick 021edf49e <Kris Kowal> fix(cosmic-swingset): Follow-up: prettier
            pick d0af526c4 <Kris Kowal> fix(cosmic-swingset): Follow-up: should publish sequences
            pick ae23c4203 <Kris Kowal> refactor: Thread solo home directory more generally (5460-publish-bundle)
            exec git branch -f 5460-publish-bundle

            pick c8090b4ab <Kris Kowal> refactor(agoric-cli): Publish with RPC instead of agd subshell
            pick 8cd36f76e <Kris Kowal> fix(casting): iterateLatest erroneously adapted getEachIterable
            pick fe113b6bc <Kris Kowal> fix(casting): Release all I/O handles between yield and next
            pick 00d84a3e1 <Kris Kowal> feat(agoric-cli): Reveal block heights to agoric follow, opt-in for lossy
            pick 2c04f46aa <Kris Kowal> chore: Update yarn.lock
            pick ea9b3946b <Kris Kowal> chore: yarn deduplicate (HEAD -> stack, 5460-publish-bundle-client)
            exec git branch -f 5460-publish-bundle-client

            # Uncomment this section to push the changes.
            # exec git push -f origin 5601-connection-refused
            # exec git push -f origin 5460-publish-bundle
            # exec git push -f origin 5460-publish-bundle-client

            # Rebase comment starts here
        "},
    }
);

testcase!(
    comment_after_instructions,
    TestCase {
        remote_name: Some("origin"),
        rebase_head_name: "feature/wip",
        branches: &[branch("feature/1", "hash1"), branch("feature/2", "hash2")],
        give: indoc! {"
            pick hash1 Implement feature 1
            pick hash2 Implement feature 2
            # Rebase instructions
        "},
        want: indoc! {"
            pick hash1 Implement feature 1
            exec git branch -f feature/1

            pick hash2 Implement feature 2
            exec git branch -f feature/2

            # Uncomment this section to push the changes.
            # exec git push -f origin feature/1
            # exec git push -f origin feature/2

            # Rebase instructions
        "},
    }
);
