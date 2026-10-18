# hikma config

Read and write configuration.

```bash
hikma config list [--json]
hikma config get <key>
hikma config set <key> <value> [--project]
hikma config unset <key> [--project]
hikma config path
hikma config add-agent <name> [--skills-dir <dir>] [--instruction-file <file>] [--project|--user] [--use] [--force] [--yes]
hikma config remove-agent <name> [--project|--user] [--yes]
```

## Keys

| Key | Values | Meaning |
|---|---|---|
| `agent` | `copilot`, `codex`, `opencode`, `claude` | Your default agent |
| `agents` | comma-separated list | Agents a project sets up; wins over `agent` |
| `registry` | `owner/repo`, git URL, or local path | Where bare skill names install from and where `push` publishes |
| `naming` | `loose` (default), `kebab-case` | Skill name rule for `create`, `push`, and `registry validate` |
| `customAgents` | set with `add-agent` | Agents you define for tools Hikma does not know; see [Custom agents](#custom-agents) |

### Config file version

Config files carry a `version` (a plain integer, currently `1`). `hikma` adds it the next time it writes a file, and a file without one is read as version 1. A file with a newer version than your `hikma` understands is refused with an error that tells you to upgrade, and it is never rewritten.

## Where values come from

From highest to lowest priority:

1. A command flag (`--agent`, `--registry`)
2. An environment variable: `HIKMA_AGENT`, `HIKMA_AGENTS`, `HIKMA_REGISTRY`, `HIKMA_NAMING`
3. Project config: `.hikma/config.json`, found from the current directory up to the repository root. Commit it to share settings.
4. User config: `hikma config path` prints the location
5. The built-in default

`hikma config list` shows every value with its source:

```text
agent     copilot  [default]
agents    claude,codex  [project]
registry  owner/repo  [user]
naming    loose  [default]
```

## Examples

```bash
hikma config set registry owner/repo                  # for you
hikma config set registry owner/repo --project        # for everyone on this repo
hikma config set agents claude,codex --project
hikma config set naming kebab-case
hikma config get registry
hikma config unset registry
```

Values are validated when set: an unknown agent or a malformed registry is rejected. Custom agent names count as known once they are defined.

## Custom agents

For a tool Hikma does not support out of the box, define where it reads skills from. `add-agent` asks, validates, and writes the definition under `customAgents`:

```text
$ hikma config add-agent acme
? Skills folder (relative to the project): .acme/skills
? Save to the project config (/path/to/repo/.hikma/config.json)? (Y/n)
? Use "acme" in this config now (add it to agents)? (Y/n)
Added agent "acme" to /path/to/repo/.hikma/config.json
  skills: .acme/skills
  agents: now includes acme
```

- If you answer no to the project file, it offers your user config, and no again cancels without writing. The question about using it defaults to yes for the project config and no for your user config, because a user-level `agents` list applies to every repository.
- The folder is checked as you type: relative, no `..`, nothing under `.git` or `.hikma`. Hikma does not create it; the first `hikma skill install --agent acme` does.
- For scripts, pass `--skills-dir`, `--project` or `--user`, and optionally `--instruction-file ACME.md` and `--use`. `--yes` accepts every default (the project config). Without a terminal the missing flag is named in the error.
- An existing definition is not replaced unless you pass `--force`. A built-in name such as `claude` is refused.
- Select it like any agent: `--agent acme`, `hikma config set agent acme`, or `hikma config set agents claude,acme --project`.
- `hikma config remove-agent acme` removes the definition and its entries in `agent` and `agents` in the same file. Installed skills and `.hikma/lock.json` are left alone.
- `hikma config list` shows custom agents after the keys, for example `agent acme: .acme/skills  [project]`.

More: [Agents and folders](/guides/agents#custom-agents).

## Naming

`naming` never affects `install`. It applies when you create or publish skills. With `kebab-case`, names may contain only lowercase letters, digits, and hyphens. Under either setting a name must be a single path segment.
