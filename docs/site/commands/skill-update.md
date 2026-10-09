# hikma skill update

Update installed skills from the source they were installed from.

```bash
hikma skill update <name>
hikma skill update --all
```

Updates use [`.hikma/lock.json`](/guides/lockfile): each skill is fetched from its recorded source and ref, which may differ from your configured registry.

## Behavior

- **Local edits stop the update.** Files are compared by hash with the lockfile. Use `--force` to overwrite your edits.
- **Changes are listed.** Added, modified, and removed files are shown. A change under `scripts/` prints a warning so you can review it.
- **Up to date.** When nothing changed, the skill is left alone.
- A skill with no lockfile entry is reported and skipped. Reinstall it with `hikma skill install --force` to track it.

## Flags

| Flag | Description |
|---|---|
| `--all` | Update every skill in the lockfile |
| `--force` | Overwrite local changes |
| `--agent <list>` | Limit to these agents' folders (not with `--all`) |

```text
$ hikma skill update deploy-helper
Updating deploy-helper from owner/repo @ 4be7f0c
  modified SKILL.md
  modified scripts/rollout.sh
  warning: scripts/ changed - review before use
```
