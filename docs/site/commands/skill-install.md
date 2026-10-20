# hikma skill install

Install a skill into the active agent's skill folder.

```bash
hikma skill install [<owner/repo>] [<name>] [flags]
```

## Forms

```bash
hikma skill install my-skill               # from your configured registry
hikma skill install owner/repo my-skill    # from any GitHub repo
hikma skill install owner/repo             # pick from the skills in that repo
hikma skill install ./local/path my-skill  # from a local checkout
hikma skill install owner/repo/skills/my-skill   # a folder in a repo
hikma skill install https://github.com/owner/repo/tree/main/skills/my-skill   # a link from the browser
hikma skill install owner/repo --skill a --skill b   # several skills from one checkout
```

A folder or a `/tree/<branch>/<folder>` link names one skill, so no name is needed. The folder has to be one hikma installs from (`skills/<name>/`, `<name>/`, or the root); for another layout, [`skill import`](/commands/skill-import) finds more. A single argument that contains a `/`, a URL, or a path is treated as a source. Anything else is a skill name looked up in your registry. See the [source list](/commands/#sources).

With a source but no name, a terminal shows a picker. Without a terminal the skills are listed and the command exits asking for a name.

## Several skills at once

Repeat `--skill` to install more than one skill from a single checkout of the source. The source is read once, every name is checked before anything is written, and one confirmation covers them all. With a registry configured the source can be left out: `hikma skill install --skill a --skill b`.

```text
Installed 2 of 3
  ok      alpha
  ok      gamma
  failed  beta   beta is already installed at .claude/skills/beta/; use --force to reinstall
```

If one skill cannot be installed the others still are, and the command exits non-zero naming the ones that failed. A name that is not in the source stops the command before it writes anything. Name skills with `--skill` or as an argument, not both.

## Flags

| Flag | Description |
|---|---|
| `--skill <name>` | A skill to install. Repeat it to install several from one checkout |
| `--ref <ref>` | Branch, tag, or commit to install from. `--pin` is an alias, matching `gh skill` |
| `--agent <list>` | Agents, comma-separated, overriding your configuration |
| `--force` | Reinstall even if already installed |
| `-g`, `--global` | Use your home directory and `~/.hikma/lock.json` instead of this project. See [Global installs](/guides/agents#global-installs) |
| `-y`, `--yes` | Skip the confirmation before changing your home directory |
| `--registry <source>` | Registry for bare names, for this command |

## What happens

1. The source and commit are printed, with where the registry setting came from.
2. The skill folder is copied into each target agent's skills folder. `.git` and symlinks are skipped.
3. The install is recorded in [`.hikma/lock.json`](/commands/sync#the-lockfile).

With more than one agent configured, the skill is installed into every distinct folder. Agents that share `.agents/skills` get one copy.

Installing does not enforce a naming convention, only that the name is a single path segment inside the skills folder.

::: warning Review what you install
Skills can contain scripts that your agent may run. Read a skill and its `scripts/` before using it. `hikma skill update` lists changed files and flags changes to `scripts/`.
:::
