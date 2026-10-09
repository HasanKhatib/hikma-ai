# hikma config

Read and write configuration.

```bash
hikma config list [--json]
hikma config get <key>
hikma config set <key> <value> [--project]
hikma config unset <key> [--project]
hikma config path
```

## Keys

| Key | Values | Meaning |
|---|---|---|
| `agent` | `copilot`, `codex`, `opencode`, `claude` | Your default agent |
| `agents` | comma-separated list | Agents a project sets up; wins over `agent` |
| `registry` | `owner/repo`, git URL, or local path | Where bare skill names install from and where `push` publishes |
| `naming` | `loose` (default), `kebab-case` | Skill name rule for `create`, `push`, and `registry validate` |

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

Values are validated when set: an unknown agent or a malformed registry is rejected.

## Naming

`naming` never affects `install`. It applies when you create or publish skills. With `kebab-case`, names may contain only lowercase letters, digits, and hyphens. Under either setting a name must be a single path segment.
