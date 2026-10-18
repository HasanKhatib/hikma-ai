# Configuration and environment

Every file hikma reads or writes, every setting, and every environment variable, in one place. See [`hikma config`](/commands/config) for the commands.

## Files

| File | Scope | Written by | Commit it? |
|---|---|---|---|
| `.hikma/config.json` | Project: found from the current directory up to the repository root | `hikma config set --project`, `hikma init` | Yes, to share settings |
| User config: `hikma config path` prints the location (`$XDG_CONFIG_HOME/hikma/config.json` or `~/.config/hikma/config.json` on Linux, `~/Library/Application Support/hikma/config.json` on macOS, `%AppData%\\hikma\\config.json` on Windows) | You, on this machine | `hikma config set` | No |
| `.hikma/lock.json` | Project lockfile | `skill install`, `update`, `remove`, `adopt` | Yes |
| `~/.hikma/lock.json` | Global lockfile, for `--global` installs | the same commands with `--global` | No |

Both config files and both lockfiles carry a `version` (a plain integer, currently `1`). A file with a newer version than your `hikma` understands is refused with an error that tells you to upgrade, and is never rewritten. A file without one is read as version 1.

Anything read from a committed file (project config, lockfile) is treated as untrusted: paths in it must stay inside the project, and nothing outside the project is written because of it.

## Settings

| Key | Values | Default | Environment variable |
|---|---|---|---|
| `agent` | `copilot`, `codex`, `opencode`, `claude`, or a custom agent name | `copilot` | `HIKMA_AGENT` |
| `agents` | comma-separated agent names; wins over `agent` | not set | `HIKMA_AGENTS` |
| `registry` | `owner/repo`, a git URL, or a local path | not set | `HIKMA_REGISTRY` |
| `naming` | `loose`, `kebab-case` | `loose` | `HIKMA_NAMING` |
| `customAgents` | an object of agent definitions, set with `hikma config add-agent` | none | none |

Values are validated when set, and again when read. `customAgents` is the only setting stored as an object:

```json
{
  "version": 1,
  "agents": ["claude", "acme"],
  "customAgents": {
    "acme": { "skillsDir": ".acme/skills", "instructionFile": "ACME.md" }
  }
}
```

## Precedence

From highest to lowest:

1. A command flag (`--agent`, `--registry`)
2. An environment variable (`HIKMA_*` above)
3. Project config
4. User config
5. The built-in default

`hikma config list` shows each value and the layer it came from.

## Other environment variables

| Variable | Effect |
|---|---|
| `NO_COLOR`, `TERM`, `COLORTERM`, `ACCESSIBLE` | Color and prompt style. See [Flags and terminal behavior](/reference/flags) |
| `HOME`, `XDG_CONFIG_HOME`, `APPDATA`, `USERPROFILE` | Where the user config and the global lockfile live |
| `GH_TOKEN`, `GITHUB_TOKEN`, `GIT_*` | Passed through to `gh` and `git`, which hikma runs for GitHub and git sources. hikma never reads or stores tokens itself |

The install script reads its own variables (`HIKMA_VERSION`, `HIKMA_INSTALL_DIR`, `HIKMA_REQUIRE_ATTESTATION`); see [Getting started](/getting-started).

An environment variable never causes a write outside the project. Anything that writes to your home directory is an explicit command that shows the target and asks first (`--yes` skips the question).
