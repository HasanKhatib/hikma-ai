# hikma skill list

List the skills in a registry or any repository.

```bash
hikma skill list [<owner/repo>] [--json]
```

Without an argument it reads the configured registry. Skills are found without an index file; see [Skill format](/guides/skill-format#where-skills-live).

```text
$ hikma skill list owner/repo
NAME                       DESCRIPTION                                         OWNER
deploy-helper              Deploy a service and check health after rollout     platform
release-notes              Draft release notes from merged pull requests       docs
```

`--json` prints the skills as JSON and sends status messages to stderr, so the output is safe to pipe.

## What is installed here

```bash
hikma skill list --installed [--json]
```

Shows the skills in this repository's agent folders and where each came from, using [`.hikma/lock.json`](/guides/lockfile):

```text
PATH                                        STATUS      SOURCE
.agents/skills/deploy-helper                ok          owner/repo @ 4be7f0c
.claude/skills/deploy-helper                modified    owner/repo @ 4be7f0c
.agents/skills/manual                       untracked   (not recorded in .hikma/lock.json)
```

| Status | Meaning |
|---|---|
| `ok` | Installed and unchanged |
| `modified` | Files differ from the lockfile (local edits) |
| `missing` | In the lockfile but the folder is gone; run [`hikma sync`](/commands/sync) |
| `untracked` | A skill folder Hikma did not install |

