# Lockfile and updates

Every install is recorded in `.hikma/lock.json`. Commit it.

```json
{
  "version": 1,
  "skills": {
    ".claude/skills/deploy-helper": {
      "name": "deploy-helper",
      "source": "owner/repo",
      "ref": "v1.2",
      "commit": "4be7f0c...",
      "files": {
        "SKILL.md": "9f2c...",
        "scripts/rollout.sh": "a1b7..."
      }
    }
  }
}
```

| Field | Meaning |
|---|---|
| key | the install folder, one entry per agent folder |
| `source` | where the skill came from |
| `ref` | the branch, tag, or commit you pinned, if any |
| `commit` | the commit that was installed |
| `files` | SHA-256 of every installed file |

The lockfile is separate from the skill. Hikma does not modify `SKILL.md`.

## Why it matters

- **Updates use the right source.** `skill update` fetches each skill from its recorded source, which can be different from your configured registry.
- **Your edits are safe.** If an installed file no longer matches its recorded hash, `update` stops and tells you. Use `--force` to overwrite.
- **Changes are visible.** An update lists added, modified, and removed files and warns when `scripts/` changed.

## Checking

`hikma skill verify` re-hashes the installed files against the lockfile and fails if anything differs, offline. `hikma sync --frozen` does the same on a fresh clone and refuses to continue on any drift. Both are meant for CI.

## Restoring

On a fresh clone, `hikma sync` installs everything in the lockfile at the recorded commits and checks the files against the recorded hashes. See [sync](/commands/sync).

## Pinning

```bash
hikma skill install owner/repo my-skill --ref v1.2
```

The ref is stored, and later updates follow that ref. A commit SHA stays at that commit.
