# hikma doctor

Check your environment and the active settings.

```bash
hikma doctor
```

```text
  ok    git installed
  ok    gh installed
  ok    gh authenticated (your-login)
  ok    agent: claude (skills in .claude/skills)
  ok    registry: owner/repo (user config)

All required checks passed.
```

## What it checks

| Check | Level |
|---|---|
| `git` is installed | required |
| `gh` is installed, signed in, and version 2 or newer | warning only. Needed for `skill push` and some private repositories. |
| The agent configuration is valid | required |
| A registry is configured | warning only. Not needed when you name a repository explicitly. |

`doctor` exits non-zero when a required check fails.
