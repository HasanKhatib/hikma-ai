# Using Hikma with gh skill

[`gh skill`](https://cli.github.com/manual/gh_skill_install) is the GitHub CLI's command for installing agent skills from GitHub repositories. Hikma and `gh skill` solve overlapping problems from different angles, and both read the same `skills/<name>/SKILL.md` layout.

## When to use which

| You want to... | Use |
|---|---|
| Browse and search public skills | `gh skill` |
| Install a public skill for many different agents | `gh skill` (it supports many more agents) |
| Set up a whole repository for several agents in one step, with shared config | `hikma init` |
| Install from a registry you named once, using bare names | `hikma` |
| Publish to your registry by pull request, with a confirmation of the target | `hikma skill push` |
| Keep a lockfile that protects local edits and flags script changes | `hikma` |
| Enforce a naming rule and validate a registry in CI | `hikma registry validate` |

## Differences to know

- **Tracking.** `gh skill` records where a skill came from inside the skill's frontmatter. Hikma keeps it in `.hikma/lock.json` and does not edit `SKILL.md`.
- **Requirements.** Hikma reads skills with `git` alone. `gh` is only needed to publish.
- **Agents.** Hikma supports Claude, Codex, Copilot, and OpenCode. `gh skill` supports many more. See [Agents and folders](/guides/agents).

## Using both

Both tools can work in the same repository, and a repository can be a source for either.

Skills that `gh skill` installed are not in `.hikma/lock.json`, so Hikma will not update, sync, or remove them on its own. It notices them, though: `hikma skill list --installed` marks them, `hikma doctor` warns, and `hikma skill update --all` lists them. To hand one over, run [`hikma skill adopt`](/commands/skill-adopt). Hikma reads the source from the metadata `gh skill` wrote, so no arguments are needed.

Going the other way, a skill installed by Hikma carries no tracking in `SKILL.md`, so `gh skill update` does not see it. Pick one tool per skill for updates.

## What gh skill does that Hikma does not

Search and preview, user-level installs (`--scope user`), and installs for many more agents. Use `gh skill` for those, and `hikma skill adopt` if you then want a lockfile.
