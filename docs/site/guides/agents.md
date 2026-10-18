# Agents and folders

Each agent reads instructions and skills from its own places. `hikma init` and `hikma skill install` write to the right ones for the agents you choose. This page records what each tool reads, so you know why.

## What each agent reads

| Agent | Instructions | Project skills folder |
|---|---|---|
| `claude` (Claude Code) | `AGENTS.md`, or `CLAUDE.md` if the repo has one | `.claude/skills` |
| `codex` | `AGENTS.md` | `.agents/skills` |
| `copilot` (GitHub Copilot) | `AGENTS.md`, `.github/copilot-instructions.md` | `.agents/skills` (also reads `.github/skills` and, in a project, `.claude/skills`; it does not read `~/.claude/skills`) |
| `opencode` | `AGENTS.md` | `.agents/skills` (also reads `.opencode/skills` and `.claude/skills`) |

Hikma uses `.agents/skills` for Codex, Copilot, and OpenCode so they share one copy, and `.claude/skills` for Claude Code.

## Claude Code and AGENTS.md

Claude Code reads `AGENTS.md` (version 2.1.277 and later) when the repository has no `CLAUDE.md`, so `hikma init --agent claude` writes only `AGENTS.md`. Things to know:

- **`CLAUDE.md` wins.** If a `CLAUDE.md` or `CLAUDE.local.md` exists in the working directory or any parent, Claude reads those and ignores `AGENTS.md`. If you add one, put `@AGENTS.md` in it to keep sharing the project rules.
- **Both files.** In Claude Code's `/config`, set **Project instructions** to load `CLAUDE.md` and `AGENTS.md` together.
- **Skills are separate.** Claude Code does not read `.agents/skills`. Skills for Claude live in `.claude/skills`.

## Choosing agents

Pick the agents your team uses. With one agent, skills go to that agent's folder. With several, `hikma skill install` writes to each distinct folder, so `claude,codex` gives you a copy in `.claude/skills` and one in `.agents/skills`. Hikma tracks both in `.hikma/lock.json` and updates them together.

Copilot and OpenCode also load skills from `.claude/skills`. If you pick one of them together with `claude`, a skill is written to both `.claude/skills` and `.agents/skills`, and that agent may use either copy. Hikma prints a warning and installs anyway. `claude,codex` does not warn, because Codex reads only `.agents/skills`.

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

- **More built-in agents.** `gh skill` supports many more. Hikma has the four above built in; add others with a [custom agent](#custom-agents).

## Global installs

By default Hikma installs into the current project. Add `--global` (`-g`) to install into your home directory instead, so a skill follows you into every project.

```bash
hikma skill install owner/repo my-skill --global
```

```text
This changes your home directory (/Users/you):
  /Users/you/.claude/skills/my-skill
  /Users/you/.hikma/lock.json
Continue? (y/N)
```

::: warning Claude personal skills do not load everywhere
Claude Code reads `~/.claude/skills` on your own machine. Personal skills do not load in cloud or Cowork sessions, so a global install does not help in a Codespace or a hosted session. Install those skills into the project instead.
:::

### Project or global

| | Project (default) | Global (`--global`) |
|---|---|---|
| Skills go in | `.claude/skills`, `.agents/skills`, ... in the repo | the same folders under your home directory |
| Lockfile | `.hikma/lock.json`, committed | `~/.hikma/lock.json`, yours alone |
| Needs a git repo | no | no |
| Asks before writing | no | yes: shows the absolute paths (`--yes` skips) |

The agent folders are the ones listed above, taken relative to your home directory. Custom agents work the same way.

`--global` works on `install`, `update`, `remove`, `adopt`, `verify`, `diff`, `sync`, and `list --installed`. `init` stays project-only.

### What stays safe

- **Explicit only.** `--global` is a flag on the command. Config, a lockfile, and environment variables cannot turn it on, so a cloned repository cannot cause a write to your home directory.
- **You see the target first.** Every global write prints the absolute paths and asks. Without a terminal it refuses until you pass `--yes`.
- **No symlinks.** Many people link `~/.claude/skills` to a dev repo of their own. Hikma will not install, update, or remove through a symlinked folder; it stops and says which one.
- **Same checks as a project.** The lockfile records hashes, `verify` and `diff` work offline, and an update that changes `scripts/` still asks.

### The same name at both scopes

When a skill name is installed in the project and globally, Hikma warns and carries on. It warns on `install`, in `list --installed` (a scope column marks both rows), in `sync`, and in `doctor`. Which copy an agent uses is the agent's decision, and most agents do not document it:

| Agent | What is known |
|---|---|
| Claude Code | [Documented](https://code.claude.com/docs/en/skills): both are listed, tagged user and project, and the slash command runs the personal copy. Not verified: which copy the model picks on its own |
| Codex | Both are listed, with no documented winner ([openai/codex#9930](https://github.com/openai/codex/issues/9930)). On 0.163.0 the project copy is listed first in 3 of 3 runs |
| OpenCode | Non-deterministic ([anomalyco/opencode#53119](https://github.com/anomalyco/opencode/issues/53119)). On 1.19.3 it lists one copy, and repeated runs of `opencode debug skill` return either one: with a project and a global copy in `.agents/skills` 11 of 12 runs returned the project copy; with project copies in `.agents/skills` and `.claude/skills` 8 returned `.agents` and 4 `.claude` |
| Copilot | Not documented. On 1.0.98 `copilot skill list` shows one copy: the project copy over the personal one (5 of 5 runs), and within a project `.agents/skills` over `.claude/skills` (3 of 3). Check yours with `copilot skill list` |

If it matters, rename one of the two. `sync` also flags a project lockfile entry that shares a name with a global skill, since a project skill with that name could inherit permissions you granted the one you trust.

### Keep your global skills in your dotfiles

`~/.hikma/lock.json` is a plain file. Put it in your dotfiles (chezmoi, stow, or a bare repo) and a fresh machine restores everything:

```bash
hikma sync --global
```

Treat this as a restore tool: it fetches each skill at its recorded commit and refuses files that do not match the recorded hashes.
