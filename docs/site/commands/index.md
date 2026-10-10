# Commands

| Command | What it does |
|---|---|
| [`hikma init`](/commands/init) | Set up a repository for one or more agents and install skills |
| [`hikma config`](/commands/config) | Read and write configuration, with the source of each value |
| [`hikma doctor`](/commands/doctor) | Check your environment and active settings |
| [`hikma registry validate`](/commands/registry-validate) | Check every skill in a registry against the format |
| [`hikma skill list`](/commands/skill-list) | List skills in a registry or any repository, or `--installed` here |
| [`hikma skill info`](/commands/skill-info) | Show details for one skill |
| [`hikma skill install`](/commands/skill-install) | Install a skill from a registry or any source |
| [`hikma skill update`](/commands/skill-update) | Update installed skills from their recorded source |
| [`hikma skill remove`](/commands/skill-remove) | Remove an installed skill |
| [`hikma skill adopt`](/commands/skill-adopt) | Track a skill that `gh skill` or another tool installed |
| [`hikma sync`](/commands/sync) | Restore every skill recorded in the lockfile |
| [`hikma skill create`](/commands/skill-create) | Scaffold a new skill |
| [`hikma skill push`](/commands/skill-push) | Publish a skill to your registry as a pull request |
| [`hikma completion`](/commands/completion) | Generate shell completions |

## Sources

Anywhere a command takes a source, it can be:

- GitHub shorthand: `owner/repo`
- a git URL: `https://host/path/repo.git` or `git@host:path/repo.git`
- a local path: `./registry` or `/abs/path`

Public sources need only `git`. For private GitHub repositories Hikma falls back to the GitHub CLI.

## Global flags

`hikma skill ...` accepts `--registry <source>` to override the configured registry for one command.

## Exit status

Commands exit non-zero when they fail. `registry validate` exits non-zero when any skill fails validation, so it works as a CI check.
