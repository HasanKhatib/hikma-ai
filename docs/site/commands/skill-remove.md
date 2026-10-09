# hikma skill remove

Remove an installed skill.

```bash
hikma skill remove <name> [--agent <list>] [--force]
```

Deletes the skill from the selected agents' skills folders and drops its entry from [`.hikma/lock.json`](/guides/lockfile). With no `--agent`, it acts on every agent configured for the repository.

```text
$ hikma skill remove deploy-helper
  removed  .claude/skills/deploy-helper
  removed  .agents/skills/deploy-helper
```

## What it protects

| Situation | Behavior |
|---|---|
| Skill is in the lockfile and unchanged | Removed |
| Skill has local edits | Kept, exits non-zero. `--force` deletes it. |
| Folder exists but is not in the lockfile | Kept, exits non-zero. `--force` deletes it. |
| Not installed for the selected agents | Error |

When the last skill is removed, the empty lockfile is deleted too.

## Flags

| Flag | Description |
|---|---|
| `--force` | Also remove untracked skills and skills with local changes |
| `--agent <list>` | Agents, comma-separated |

To bring a removed skill back, run [`hikma skill install`](/commands/skill-install) again.
