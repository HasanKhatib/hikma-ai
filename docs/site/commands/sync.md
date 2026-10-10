# hikma sync

Restore every skill recorded in `.hikma/lock.json`.

```bash
hikma sync [--yes] [--force] [--dry-run] [--frozen]
```

Use it on a fresh clone to get exactly the skills the repository was committed with, the way `npm ci` restores dependencies from a lockfile.

```text
$ hikma sync
Skills recorded in .hikma/lock.json: 2
  restore   .claude/skills/deploy-helper  (owner/repo @ 4be7f0c)
  ok        .agents/skills/deploy-helper  (owner/repo @ 4be7f0c)
? Fetch 1 skill(s) from 1 source(s) listed above? (y/N) y
Fetching owner/repo @ 4be7f0c...
  restored .claude/skills/deploy-helper

Done: 1 restored, 1 up to date, 0 skipped, 0 failed.
```

## What it does

- Fetches each skill from its **recorded source at its recorded commit**, not the latest version. Use [`skill update`](/commands/skill-update) to move forward.
- **Verifies the files** against the hashes in the lockfile. Content that does not match is refused and nothing is installed.
- Leaves skills that are already installed and unchanged alone.
- Skips skills you edited locally and exits non-zero, so you notice. `--force` overwrites them.
- Works without a configured registry; it only uses the sources in the lockfile.

## Safety

The lockfile comes from the repository you cloned, so `sync` treats it as untrusted:

- It lists the skills and sources it will fetch and **asks you to confirm**. `--yes` skips the prompt, and is required without a terminal. When nothing needs fetching it does not ask.
- Entries whose install path is outside the agent skill folders (`.claude/skills/<name>`, `.agents/skills/<name>`) are **refused** before anything is written.
- Review what you restore: skills can contain scripts your agent may run.

## Flags

| Flag | Description |
|---|---|
| `-y`, `--yes` | Skip the confirmation prompt |
| `--force` | Overwrite skills that have local changes |
| `--dry-run` | Show the plan without changing anything |
| `--frozen` | Fail on any drift instead of skipping it. For CI |

## In CI

```bash
hikma sync --frozen --yes
```

`--frozen` must reproduce the lockfile exactly, and fails before changing anything when it cannot:

- a skill has local changes (`--force` overwrites them instead)
- an entry is not pinned to a commit
- a source cannot be fetched at its recorded commit, or its files do not match the recorded hashes

Without `--frozen`, `hikma sync --yes` skips skills with local changes and exits non-zero. To check without fetching anything, use [`hikma skill verify`](/commands/skill-verify).
