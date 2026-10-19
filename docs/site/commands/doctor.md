# hikma doctor

Check your environment and the active settings.

```bash
hikma doctor
```

```text
  ok    gh installed
  ok    gh version (2.92.0)
  ok    git installed
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
| A GitHub registry is set | `ok`, and a line saying how `skill push` and `skill import` will publish to it (commit to the default branch, a pull request, or through your fork). Printed only when `gh` can tell; see [How publishing decides](/guides/registry#how-publishing-decides) |
| A skill name is installed in both the project and your home directory | warning only. Agents differ on which copy they use; see [Global installs](/guides/agents#global-installs). |

A warning prints the reason and a `fix:` line, and does not fail the check. `doctor` exits non-zero when a required check fails.
