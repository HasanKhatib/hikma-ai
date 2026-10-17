# hikma init

Set up a repository for one or more AI agents.

```bash
hikma init [flags]
```

`init` writes each selected agent's instruction files, records the agents (and an optional registry) in `.hikma/config.json` so teammates share them, and can install skills into every selected agent's skills folder. `hikma scaffold` is an alias.

## Examples

```bash
# Claude and Codex, with a registry and one skill
hikma init --agent claude,codex --registry owner/repo --skill my-skill

# Preview without changing anything
hikma init --agent claude --dry-run --project-name demo

# Non-interactive (CI): all values from flags
hikma init --agent claude --project-name demo --owner platform --technology Go
```

## Flags

| Flag | Description |
|---|---|
| `--agent <list>` | Agents, comma-separated: `copilot`, `codex`, `opencode`, `claude`. Prompts when omitted in a terminal. |
| `--registry <source>` | Project registry: `owner/repo`, a git URL, or a local path. Saved to `.hikma/config.json`. |
| `--skill <name>` | Install this skill from the registry. Repeatable. |
| `--path <dir>` | Target directory (default: current directory). `--skill` requires running in the target directory. |
| `--force` | Overwrite existing files and reinstall skills. |
| `--dry-run` | Print what would happen without changing anything. |
| `--project-name`, `--owner`, `--technology` | Fill the generated instructions without prompting. |

## What gets written

| Agent | Instruction files | Skills folder |
|---|---|---|
| `claude` | `AGENTS.md` | `.claude/skills/` |
| `codex`, `copilot`, `opencode` | `AGENTS.md` | `.agents/skills/` |

Agents that share a folder are written once. Claude Code reads `AGENTS.md` when the repo has no `CLAUDE.md`. See [Agents and folders](/guides/agents) for why each agent gets the folder it does.

Existing files are skipped unless you pass `--force`. Without a terminal, `init` runs non-interactively using flags and defaults.

## Project config

`init` saves the chosen agents (and registry, when given) in `.hikma/config.json`:

```json
{
  "agents": ["claude", "codex"],
  "registry": "owner/repo"
}
```

Commit it. Later `hikma skill install` and `update` calls in that repository act on every listed agent without extra flags. See [Share it with your team](#share-it-with-your-team).

## Share it with your team

Commit `AGENTS.md`, `.hikma/config.json`, `.hikma/lock.json`, and the skills folders so everyone who clones the repository gets the same instructions and skills.

### What teammates get

After cloning, `hikma skill install other-skill` installs into every agent listed in `.hikma/config.json`, from the project's registry, with no flags. Their personal `agent` default does not override a project's `agents` list.

### On a fresh clone

If the skills folders are not committed (for example they are git-ignored), restore them from the lockfile:

```bash
hikma sync
```

### Updating

```bash
hikma skill update --all
```

Review the printed changes, especially anything under `scripts/`, and commit the result. See [The lockfile](/commands/sync#the-lockfile).

### Edit the instructions

`AGENTS.md` is generated once with placeholders for build commands and invariants. Fill those in; Hikma will not overwrite the file unless you pass `--force`.

### Settings you can share

| Setting | Command |
|---|---|
| Agents | `hikma config set agents claude,codex --project` |
| Registry | `hikma config set registry owner/repo --project` |
| Naming rule | `hikma config set naming kebab-case --project` |
