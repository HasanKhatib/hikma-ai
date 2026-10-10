# hikma skill update

Update installed skills from the source they were installed from.

```bash
hikma skill update <name>
hikma skill update --all
```

Updates use [`.hikma/lock.json`](/guides/lockfile): each skill is fetched from its recorded source and ref, which may differ from your configured registry.

## Behavior

- **Local edits stop the update.** Files are compared by hash with the lockfile. Use `--force` to overwrite your edits.
- **Changes are listed.** Added, modified, and removed files are shown.
- **Script changes need confirmation.** An update that changes `scripts/` prints a warning and asks before applying. Without a terminal it is refused until you pass `--yes`. Review it first with [`hikma skill diff`](/commands/skill-diff).
- **Up to date.** When nothing changed, the skill is left alone.
- A skill with no lockfile entry is reported and skipped. Track it with [`hikma skill adopt`](/commands/skill-adopt).
- To preview an update first, run [`hikma skill diff`](/commands/skill-diff).

## Flags

| Flag | Description |
|---|---|
| `--all` | Update every skill in the lockfile |
| `--force` | Overwrite local changes |
| `-y`, `--yes` | Apply updates that change `scripts/` without asking |
| `--agent <list>` | Limit to these agents' folders (not with `--all`) |

```text
$ hikma skill update deploy-helper
Updating deploy-helper from owner/repo @ 4be7f0c
  modified SKILL.md
  modified scripts/rollout.sh
  warning: scripts/ changed - review before use
```
