# Agents and folders

Each agent reads instructions and skills from its own places. `hikma init` and `hikma skill install` write to the right ones for the agents you choose. This page records what each tool reads, so you know why.

## What each agent reads

| Agent | Instructions | Project skills folder |
|---|---|---|
| `claude` (Claude Code) | `AGENTS.md`, or `CLAUDE.md` if the repo has one | `.claude/skills` |
| `codex` | `AGENTS.md` | `.agents/skills` |
| `copilot` (GitHub Copilot) | `AGENTS.md`, `.github/copilot-instructions.md` | `.agents/skills` (also reads `.github/skills` and `.claude/skills`) |
| `opencode` | `AGENTS.md` | `.agents/skills` (also reads `.opencode/skills` and `.claude/skills`) |

Hikma uses `.agents/skills` for Codex, Copilot, and OpenCode so they share one copy, and `.claude/skills` for Claude Code.

## Claude Code and AGENTS.md

Claude Code reads `AGENTS.md` (version 2.1.277 and later) when the repository has no `CLAUDE.md`, so `hikma init --agent claude` writes only `AGENTS.md`. Things to know:

- **`CLAUDE.md` wins.** If a `CLAUDE.md` or `CLAUDE.local.md` exists in the working directory or any parent, Claude reads those and ignores `AGENTS.md`. If you add one, put `@AGENTS.md` in it to keep sharing the project rules.
- **Both files.** In Claude Code's `/config`, set **Project instructions** to load `CLAUDE.md` and `AGENTS.md` together.
- **Skills are separate.** Claude Code does not read `.agents/skills`. Skills for Claude live in `.claude/skills`.

## Choosing agents

Pick the agents your team uses. With one agent, skills go to that agent's folder. With several, `hikma skill install` writes to each distinct folder, so `claude,codex` gives you a copy in `.claude/skills` and one in `.agents/skills`. Hikma tracks both in `.hikma/lock.json` and updates them together.

Copilot and OpenCode also load skills from `.claude/skills`. If you pick one of them together with `claude`, a skill is written to both `.claude/skills` and `.agents/skills`, and that agent may list it twice. Hikma prints a warning and installs anyway. `claude,codex` does not warn, because Codex reads only `.agents/skills`.

If someone on a Claude project also tries Codex, run `hikma skill install` with `--agent codex` for that install; the skill lands in `.agents/skills`.

## Custom agents

If your agent is not one of the four above, describe where it reads from. Run `hikma config add-agent acme` (see [`hikma config`](/commands/config#custom-agents)), or add a `customAgents` entry to `.hikma/config.json` (or your user config) yourself. Then use its name like any other agent:

```json
{
  "agents": ["claude", "acme"],
  "customAgents": {
    "acme": { "skillsDir": ".acme/skills", "instructionFile": "ACME.md" }
  }
}
```

```bash
hikma init --agent claude,acme
hikma skill install code-review --agent acme
```

Check your tool's own documentation for the right folder and instruction file. Rules, enforced when the config loads:

- `skillsDir` is required. `instructionFile` is optional; `hikma init` writes the same starter instructions as `AGENTS.md` under that name, and never overwrites an existing file unless you pass `--force`.
- Both are relative, forward-slash paths inside the project. No `..`, no absolute paths or drive letters, and nothing under `.git` or `.hikma`.
- A name is lowercase letters, digits, and hyphens, and cannot reuse a built-in agent name.
- If a project and your user config define the same name, the project wins.
- `hikma sync` restores skills in a custom folder only while the agent is defined in config. `hikma skill install --force` and `hikma sync --force` refuse to replace an existing folder that has no `SKILL.md`, so a custom agent can never make Hikma delete project code.

`hikma config set` cannot edit definitions; use `add-agent` and `remove-agent`, or edit the file. You select one by name like any other agent: `hikma config set agent acme` or `hikma config set agents claude,acme`.

## The Agent Skills format

All of these tools use the open [Agent Skills](https://agentskills.io/specification) format: a folder with a `SKILL.md`, plus optional `scripts/`, `references/`, and `assets/`. See [Skill format](/guides/skill-format) for the rules Hikma checks.

Some agents add their own files next to `SKILL.md`, such as Codex's optional `agents/openai.yaml` for UI and invocation settings. Hikma copies and tracks these like any other file.

## Not covered

- **User-level installs** (`~/.claude/skills`, `~/.agents/skills`). Use [`gh skill`](/guides/gh-skill) with `--scope user`.
- **More built-in agents.** `gh skill` supports many more. Hikma has the four above built in; add others with a [custom agent](#custom-agents).
