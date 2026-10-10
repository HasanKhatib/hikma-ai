# hikma skill verify

Check that installed skills still match the hashes in `.hikma/lock.json`.

```bash
hikma skill verify [<name>] [--agent <list>]
```

It re-hashes the installed files and compares them with the [lockfile](/guides/lockfile). It works offline, changes nothing, and exits non-zero when any skill differs, so it fits a CI check or a pre-commit hook. Without a name it checks every skill in the lockfile.

```text
$ hikma skill verify
  FAIL  .claude/skills/deploy-helper
          modified scripts/deploy.sh
          added    notes.md
          warning: scripts/ differ from what was installed
  ok    .agents/skills/deploy-helper

2 skill(s) checked, 1 differ.
```

A skill whose folder is missing is reported too; [`hikma sync`](/commands/sync) restores it.

## Verify, diff, or sync

| You want to | Use |
|---|---|
| Know whether anything changed since install | `skill verify` |
| See what changed, and what an update would bring | [`skill diff`](/commands/skill-diff) |
| Put the files back as recorded | [`sync`](/commands/sync) |
