# Global installs

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

## Project or global

| | Project (default) | Global (`--global`) |
|---|---|---|
| Skills go in | `.claude/skills`, `.agents/skills`, ... in the repo | the same folders under your home directory |
| Lockfile | `.hikma/lock.json`, committed | `~/.hikma/lock.json`, yours alone |
| Needs a git repo | no | no |
| Asks before writing | no | yes: shows the absolute paths (`--yes` skips) |

The agent folders are the ones from [Agents and folders](/guides/agents), taken relative to your home directory. Custom agents work the same way.

`--global` works on `install`, `update`, `remove`, `adopt`, `verify`, `diff`, `sync`, and `list --installed`. `init` stays project-only.

## What stays safe

- **Explicit only.** `--global` is a flag on the command. Config, a lockfile, and environment variables cannot turn it on, so a cloned repository cannot cause a write to your home directory.
- **You see the target first.** Every global write prints the absolute paths and asks. Without a terminal it refuses until you pass `--yes`.
- **No symlinks.** Many people link `~/.claude/skills` to a dev repo of their own. Hikma will not install, update, or remove through a symlinked folder; it stops and says which one.
- **Same checks as a project.** The lockfile records hashes, `verify` and `diff` work offline, and an update that changes `scripts/` still asks.

## The same name at both scopes

When a skill name is installed in the project and globally, Hikma warns and carries on. It warns on `install`, in `list --installed` (a scope column marks both rows), in `sync`, and in `doctor`. Which copy an agent uses is the agent's decision, and most agents do not document it:

| Agent | What is known |
|---|---|
| Claude Code | [Documented](https://code.claude.com/docs/en/skills): both are listed, tagged user and project, and the slash command runs the personal copy. Not verified: which copy the model picks on its own |
| Codex | Both are listed, with no documented winner ([openai/codex#9930](https://github.com/openai/codex/issues/9930)) |
| OpenCode | Reported as non-deterministic: the last file read wins ([anomalyco/opencode#53119](https://github.com/anomalyco/opencode/issues/53119)) |
| Copilot | Not documented. Check with `/skills info NAME` |

If it matters, rename one of the two. `sync` also flags a project lockfile entry that shares a name with a global skill, since a project skill with that name could inherit permissions you granted the one you trust.

## Keep your global skills in your dotfiles

`~/.hikma/lock.json` is a plain file. Put it in your dotfiles (chezmoi, stow, or a bare repo) and a fresh machine restores everything:

```bash
hikma sync --global
```

Treat this as a restore tool: it fetches each skill at its recorded commit and refuses files that do not match the recorded hashes.
