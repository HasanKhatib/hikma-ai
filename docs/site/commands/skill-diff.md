# hikma skill diff

See what differs for an installed skill, without changing anything.

```bash
hikma skill diff <name> [--patch] [--agent <list>]
```

The skill must be recorded in [`.hikma/lock.json`](/guides/lockfile). If `gh skill` installed it, run [`hikma skill adopt`](/commands/skill-adopt) first.

## What it shows

- **Local edits.** Files that differ from what was installed: added, modified, or removed.
- **Upstream changes.** What [`hikma skill update`](/commands/skill-update) would bring in from the recorded source and ref. A change under `scripts/` prints a warning.
- **Both sides.** A file changed locally and upstream is marked `(also edited locally)`, because an update would overwrite your edits to it.

```text
$ hikma skill diff deploy-helper
.claude/skills/deploy-helper (owner/repo @ 4be7f0c)
Local edits:
  modified SKILL.md
Upstream changes (owner/repo @ a91c2d7):
  modified scripts/deploy.sh
  modified SKILL.md  (also edited locally)
  warning: scripts/ changed - review before updating
```

## Flags

| Flag | Description |
|---|---|
| `--patch` | Print the contents that differ (installed against upstream). Needs `git`. |
| `--agent <list>` | Limit to these agents' folders |

Nothing is written to your skill folders or the lockfile. Use it to review a script change before you run `hikma skill update`.
