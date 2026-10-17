# hikma skill adopt

Track a skill that another tool installed, so Hikma can update, sync, and remove it.

```bash
hikma skill adopt <name> [--source <owner/repo>] [--ref <ref>] [--agent <list>] [--global]
```

## Skills installed by gh skill

[`gh skill`](/commands/skill-adopt#using-hikma-with-gh-skill) writes where a skill came from into its `SKILL.md` (`github-repo`, `github-ref`, `github-tree-sha`, and `github-pinned` when installed with `--pin`). Hikma reads that, so adopting needs no arguments:

```text
$ hikma skill list --installed
PATH                  STATUS      SOURCE
.claude/skills/pdf    untracked   installed by gh skill from https://github.com/anthropics/skills; run: hikma skill adopt pdf

$ hikma skill adopt pdf
  adopted  .claude/skills/pdf  (anthropics/skills @ dbd4588)
```

Adopt checks that the source still has a skill with that name, then records the source, the commit, and the hashes of the files as they are on disk. Nothing is downloaded into the folder. The next `hikma skill update` replaces the `gh skill` metadata in `SKILL.md` with the registry's version.

A pinned skill (`github-pinned`) is adopted at its pinned version.

## Any other skill

For a skill without `gh skill` metadata, say where it came from:

```bash
hikma skill adopt my-skill --source owner/repo --ref v1.2.0
```

## Flags

| Flag | Description |
|---|---|
| `--source <source>` | `owner/repo`, git URL, or path. Default: read from `gh skill` metadata. |
| `--ref <ref>` | Branch, tag, or commit to update from |
| `--agent <list>` | Which agents' folders to adopt from |
| `-g`, `--global` | Adopt a skill from your home directory into `~/.hikma/lock.json` |
| `-y`, `--yes` | Skip the confirmation before changing your home directory |

## A skill with no gh skill metadata

If the skill was not installed by `gh skill`, Hikma cannot tell where it came from and says so, with the commands to run. Pass `--source <owner/repo>` if it came from a repository (the message fills in your configured registry when there is one), or run [`hikma skill push`](/commands/skill-push) if it is your own skill and not published yet.

## Where Hikma points you here

- `hikma skill list --installed` marks these skills.
- `hikma doctor` warns about them.
- `hikma skill update --all` mentions them at the end.
- `hikma skill update <name>` on one stops and tells you to adopt it.

## Using Hikma with gh skill

[`gh skill`](https://cli.github.com/manual/gh_skill_install) is the GitHub CLI's command for installing agent skills from GitHub repositories. Hikma and `gh skill` solve overlapping problems from different angles, and both read the same `skills/<name>/SKILL.md` layout.

### When to use which

| You want to... | Use |
|---|---|
| Browse and search public skills | `gh skill` |
| Install a public skill for many different agents | `gh skill` (it supports many more agents) |
| Set up a whole repository for several agents in one step, with shared config | `hikma init` |
| Install from a registry you named once, using bare names | `hikma` |
| Publish to your registry, with a confirmation of the target | `hikma skill push` |
| Keep a lockfile that protects local edits and flags script changes | `hikma` |
| Enforce a naming rule and validate a registry in CI | `hikma registry validate` |

### Differences to know

- **Tracking.** `gh skill` records where a skill came from inside the skill's frontmatter. Hikma keeps it in `.hikma/lock.json` and does not edit `SKILL.md`.
- **Requirements.** Hikma reads skills with `git` alone. `gh` is only needed to publish.
- **Agents.** Hikma supports Claude, Codex, Copilot, and OpenCode. `gh skill` supports many more. See [Agents and folders](/guides/agents).

### Using both

Both tools can work in the same repository, and a repository can be a source for either.

Skills that `gh skill` installed are not in `.hikma/lock.json`, so Hikma will not update, sync, or remove them on its own. It notices them, though: `hikma skill list --installed` marks them, `hikma doctor` warns, and `hikma skill update --all` lists them. To hand one over, run [`hikma skill adopt`](/commands/skill-adopt). Hikma reads the source from the metadata `gh skill` wrote, so no arguments are needed.

Going the other way, a skill installed by Hikma carries no tracking in `SKILL.md`, so `gh skill update` does not see it. Pick one tool per skill for updates.

### What gh skill does that Hikma does not

Search and preview, and installs for many more agents. Use `gh skill` for those, and `hikma skill adopt` if you then want a lockfile. For user-level installs, either tool works: `gh skill --scope user`, or `hikma skill install --global` for a lockfile, `verify`, and `sync` ([Global installs](/guides/agents#global-installs)).
