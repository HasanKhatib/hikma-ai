# Troubleshooting

The messages people hit most, and what to do. `hikma doctor` checks your environment and names anything missing.

## `hikma: command not found`

The folder `hikma` was installed into is not on your `PATH`. The install script uses `~/.local/bin`.

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc    # zsh; use ~/.bashrc for bash
```

Open a new terminal and run `hikma --version`. To install somewhere else, set `HIKMA_INSTALL_DIR` before running the script.

## `skill push` says `gh` is missing or not signed in

Publishing uses the GitHub CLI. Install it, then sign in once:

```bash
gh auth login
```

## `skill 'x' is not at …, but it is at …`

The skill is in another agent's folder than the one Hikma is set to read. The message names the `--agent` that finds it:

```bash
hikma skill push x --agent claude
```

The [menu](/guides/menu) does this for you when you publish.

## `has local changes; use --force`

You edited an installed skill, and an install or update would replace your edits. Look first:

```bash
hikma skill diff <name>
```

Then keep your edits (do nothing), publish them with [`skill push`](/commands/skill-push), or replace them with `--force`.

## `skill verify` prints `FAIL`

An installed skill no longer matches `.hikma/lock.json`: a file was edited, added, or removed. [`skill diff`](/commands/skill-diff) shows which. [`sync`](/commands/sync) puts the files back as recorded.

## GitHub is rate limiting requests

Pushing uploads one file at a time. A rate-limited call is retried after a minute, then two, then four. If it still fails, nothing was committed, so run the command again later.

## `hikma upgrade` says Homebrew is behind

A release is published before the Homebrew formula catches up. Wait a few minutes and run `hikma upgrade` again.

## `hikma` prints help instead of opening the menu

The menu opens only when both input and output are a terminal. In a pipe, a script, or with `--no-input`, bare `hikma` prints help.
