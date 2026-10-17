# hikma skill adopt

Track a skill that another tool installed, so Hikma can update, sync, and remove it.

```bash
hikma skill adopt <name> [--source <owner/repo>] [--ref <ref>] [--agent <list>] [--global]
```

## Skills installed by gh skill

[`gh skill`](/guides/gh-skill) writes where a skill came from into its `SKILL.md` (`github-repo`, `github-ref`, `github-tree-sha`, and `github-pinned` when installed with `--pin`). Hikma reads that, so adopting needs no arguments:

```text
$ hikma skill list --installed
PATH                  STATUS      SOURCE
.claude/skills/pdf    untracked   installed by gh skill from https://github.com/anthropics/skills; run: hikma skill adopt pdf

$ hikma skill adopt pdf
  adopted  .claude/skills/pdf  (anthropics/skills @ dbd4588)
```

Adopt checks that the source still has a skill with that name, then records the source, the commit, and the hashes of the files as they are on disk. Nothing is downloaded into the folder. The next `hikma skill update` replaces the `gh skill` metadata in `SKILL.md` with the registry's version.

A pinned skill (`github-pinned`) is adopted at its pinned version.

## Any other skill

For a skill without `gh skill` metadata, say where it came from:

```bash
hikma skill adopt my-skill --source owner/repo --ref v1.2.0
```

## Flags

| Flag | Description |
|---|---|
| `--source <source>` | `owner/repo`, git URL, or path. Default: read from `gh skill` metadata. |
| `--ref <ref>` | Branch, tag, or commit to update from |
| `--agent <list>` | Which agents' folders to adopt from |
| `-g`, `--global` | Adopt a skill from your home directory into `~/.hikma/lock.json` |
| `-y`, `--yes` | Skip the confirmation before changing your home directory |

## A skill with no gh skill metadata

If the skill was not installed by `gh skill`, Hikma cannot tell where it came from and says so, with the commands to run. Pass `--source <owner/repo>` if it came from a repository (the message fills in your configured registry when there is one), or run [`hikma skill push`](/commands/skill-push) if it is your own skill and not published yet.

## Where Hikma points you here

- `hikma skill list --installed` marks these skills.
- `hikma doctor` warns about them.
- `hikma skill update --all` mentions them at the end.
- `hikma skill update <name>` on one stops and tells you to adopt it.
