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
```

A single argument that contains a `/`, a URL, or a path is treated as a source. Anything else is a skill name looked up in your registry. See the [source list](/commands/#sources).

With a source but no name, a terminal shows a picker. Without a terminal the skills are listed and the command exits asking for a name.

## Flags

| Flag | Description |
|---|---|
| `--ref <ref>` | Branch, tag, or commit to install from |
| `--agent <list>` | Agents, comma-separated, overriding your configuration |
| `--force` | Reinstall even if already installed |
| `--registry <source>` | Registry for bare names, for this command |

## What happens

1. The source and commit are printed, with where the registry setting came from.
2. The skill folder is copied into each target agent's skills folder. `.git` and symlinks are skipped.
3. The install is recorded in [`.hikma/lock.json`](/guides/lockfile).

With more than one agent configured, the skill is installed into every distinct folder. Agents that share `.agents/skills` get one copy.

Installing does not enforce a naming convention, only that the name is a single path segment inside the skills folder.

::: warning Review what you install
Skills can contain scripts that your agent may run. Read a skill and its `scripts/` before using it. `hikma skill update` lists changed files and flags changes to `scripts/`.
:::
