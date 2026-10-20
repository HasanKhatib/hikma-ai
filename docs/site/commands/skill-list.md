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

Skills you already have installed for your agent are marked `[installed]` in front of the name. When none is, the table has no marker column. `--json` carries no markers.

`--json` prints `{ "version": 1, "skills": [...] }` and sends status messages to stderr, so the output is safe to pipe. See [Exit codes and JSON](/reference/output).

## What is installed here

```bash
hikma skill list --installed [--global] [--json]
```

Shows the skills in this repository's agent folders and in your home directory, and where each came from, using [`.hikma/lock.json`](/commands/sync#the-lockfile) and `~/.hikma/lock.json`. A name installed at both scopes is marked, because agents differ on which copy they use ([Global installs](/guides/agents#global-installs)). `--global` lists only your home directory's skills. With `--json`, the document is `{ "version": 1, "skills": [...] }` with a `status` of `ok`, `modified`, `missing`, or `untracked`.

```text
SCOPE     PATH                                        STATUS      SOURCE
project   .agents/skills/deploy-helper                ok          owner/repo @ 4be7f0c
project   .claude/skills/deploy-helper                modified    owner/repo @ 4be7f0c
project   .agents/skills/manual                       untracked   (not recorded in .hikma/lock.json)
global    .claude/skills/deploy-helper                ok          owner/repo @ 4be7f0c  [also installed at project scope]
```

| Status | Meaning |
|---|---|
| `ok` | Installed and unchanged |
| `modified` | Files differ from the lockfile (local edits) |
| `missing` | In the lockfile but the folder is gone; run [`hikma sync`](/commands/sync) |
| `untracked` | A skill folder Hikma did not install |

