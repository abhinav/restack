# restack [![Go](https://github.com/abhinav/restack/actions/workflows/go.yml/badge.svg)](https://github.com/abhinav/restack/actions/workflows/go.yml)

restack augments the experience of performing an interactive Git rebase to make
it more friendly to workflows that involve lots of interdependent branches.

For more background on why this exists and the workflow it facilitates, see
[Automatically Restacking Git Branches][1].

  [1]: https://abhinavg.net/posts/restacking-branches/

## Installation

Use one of the following options to install restack.

- If you use **Homebrew** on macOS or **Linuxbrew** on Linux, run the following
  command to download and install a pre-built binary.

    ```
    brew install abhinav/tap/restack
    ```
    
- If you use **ArchLinux**, install it from AUR
  using the [restack-bin package] package for a pre-built binary
  or the [restack package] package to build it from source.

    ```
    git clone https://aur.archlinux.org/restack-bin.git
    cd restack-bin
    makepkg -si
    ```

  With an AUR helper like [yay], run the following instead:

    ```
    yay -S restack-bin # pre-built binary
    yay -S restack     # build from source
    ```

- Download a pre-built binary from the [GitHub Releases] page and place it on
  your `$PATH`.

- Build it from source if you have the Go toolchain installed.

    ```
    go install github.com/abhinav/restack/cmd/restack@latest
    ```

  [restack-bin package]: https://aur.archlinux.org/packages/restack-bin
  [restack package]: https://aur.archlinux.org/packages/restack
  [yay]: https://github.com/Jguer/yay
  [GitHub Releases]: https://github.com/abhinav/restack/releases

## Setup

restack works by installing itself as a Git [`sequence.editor`].
You can set this up manually or let restack do it for you automatically.

  [`sequence.editor`]: https://git-scm.com/docs/git-config#Documentation/git-config.txt-sequenceeditor

### Automatic Setup

Run `restack setup` to configure `git` to use `restack`.

    restack setup

### Manual Setup

If you would rather not have restack change your `.gitconfig`,
you can set it up manually by running:

```
git config sequence.editor "restack edit"
```

See `restack edit --help` for the different options accepted by `restack edit`.

## Usage

restack automatically recognizes branches being touched by the rebase and adds
rebase instructions which update these branches as their heads move.

The generated instruction list also includes an opt-in commented-out section
that will push these branches to the remote.

For example, given,

    o master
     \
      o A
      |
      o B (feature1)
       \
        o C
        |
        o D (feature2)
         \
          o E
          |
          o F
          |
          o G (feature3)
           \
            o H (feature4, HEAD)

Running `git rebase -i master` from branch `feature4` will give you the
following instruction list.

    pick A
    pick B
    exec git branch -f feature1

    pick C
    pick D
    exec git branch -f feature2

    pick E
    pick F
    pick G
    exec git branch -f feature3

    pick H

    # Uncomment this section to push the changes.
    # exec git push -f origin feature1
    # exec git push -f origin feature2
    # exec git push -f origin feature3

So any changes made before each `exec git branch -f` will become part of that
branch and all following changes will be made on top of that.

## Credits

Thanks to [@kriskowal] for the initial implementation of this tool as a
script.

  [@kriskowal]: https://github.com/kriskowal
  
## FAQ

### Can I make restacking opt-in?

If you don't want restack to do its thing on every `git rebase`,
you can make it opt-in by introducing a new Git command.

To do this, first make sure you don't have restack set up for the regular
`git rebase`:

```
git config --global --unset sequence.editor
```

Next, create a file named `git-restack` with the following contents.

```bash
#!/bin/bash
exec git -c sequence.editor="restack edit" rebase "$@"
```

Mark it as executable and place it somewhere on `$PATH`.

```
chmod +x git-restack
mv git-restack ~/bin/git-restack
```

Going forward, you can run `git rebase` for a plain rebase,
and `git restack` to run a rebase with support for branch restacking.
