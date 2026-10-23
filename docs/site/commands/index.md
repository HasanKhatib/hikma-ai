# Commands

## Set up

| Command | What it does |
|---|---|
| [`hikma init`](/commands/init) | Set up a repository for one or more agents and install skills |
| [`hikma config`](/commands/config) | Read and write configuration, with the source of each value |
| [`hikma doctor`](/commands/doctor) | Check your environment and active settings |

## Find and install skills

| Command | What it does |
|---|---|
| [`hikma skill list`](/commands/skill-list) | List skills in a registry or any repository, or `--installed` here |
| [`hikma skill info`](/commands/skill-info) | Show details for one skill |
| [`hikma skill install`](/commands/skill-install) | Install a skill from a registry or any source |

## Keep skills current

| Command | What it does |
|---|---|
| [`hikma skill update`](/commands/skill-update) | Update installed skills from their recorded source |
| [`hikma skill verify`](/commands/skill-verify) | Check installed skills against the lockfile hashes |
| [`hikma skill diff`](/commands/skill-diff) | Show local edits and upstream changes for a skill |
| [`hikma skill remove`](/commands/skill-remove) | Remove an installed skill |
| [`hikma skill adopt`](/commands/skill-adopt) | Track a skill that `gh skill` or another tool installed |
| [`hikma sync`](/commands/sync) | Restore every skill recorded in the lockfile |

## Create and publish

| Command | What it does |
|---|---|
| [`hikma skill create`](/commands/skill-create) | Scaffold a new skill |
| [`hikma skill push`](/commands/skill-push) | Publish a skill to your registry |
| [`hikma skill import`](/commands/skill-import) | Bring a skill you found anywhere into your registry, after a review |
| [`hikma registry validate`](/commands/registry-validate) | Check every skill in a registry against the format |

## Other

| Command | What it does |
|---|---|
| [`hikma completion`](/commands/completion) | Generate shell completions |
| [`hikma upgrade`](/commands/upgrade) | Upgrade hikma itself, for the way you installed it |

Run `hikma` with no arguments for a [menu](/guides/menu) of the common tasks.

## Sources

Anywhere a command takes a source, it can be:

- GitHub shorthand: `owner/repo`
- a folder in a GitHub repo: `owner/repo/skills/name`, or a `https://github.com/owner/repo/tree/<branch>/<folder>` link (`install` and `import`)
- a git URL: `https://host/path/repo.git` or `git@host:path/repo.git`
- a local path: `./registry` or `/abs/path`

Public sources need only `git`. For private GitHub repositories Hikma falls back to the GitHub CLI.

## Global flags

`--no-input` and `--no-color` work on every command; see [Flags and terminal behavior](/reference/flags). `hikma skill ...` accepts `--registry <source>` to override the configured registry for one command.

## Exit status

Commands exit `0` on success, `1` on failure, `2` for wrong usage, and `3` when installed skills differ from the lockfile. `registry validate` exits non-zero when any skill fails validation, so it works as a CI check. See [Exit codes and JSON](/reference/output).
