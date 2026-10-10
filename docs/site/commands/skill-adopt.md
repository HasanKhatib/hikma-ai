# hikma skill adopt

Track a skill that another tool installed, so Hikma can update, sync, and remove it.

```bash
hikma skill adopt <name> [--source <owner/repo>] [--ref <ref>] [--agent <list>]
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

## Where Hikma points you here

- `hikma skill list --installed` marks these skills.
- `hikma doctor` warns about them.
- `hikma skill update --all` mentions them at the end.
- `hikma skill update <name>` on one stops and tells you to adopt it.
