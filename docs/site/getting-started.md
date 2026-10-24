---
description: "Install Hikma with Homebrew or a script, set up a project, and follow the path you need: install skills, publish your own, or import one."
---

# Getting started

This page gets Hikma installed, then points you to the path you need.

New to the words? A **skill** is a folder of instructions an AI agent reads. A **registry** is a Git repository where your skills live. The **lockfile** (`.hikma/lock.json`) is the list of what is installed. The [glossary](/guides/glossary) has the rest.

## 1. Install

::: code-group

```bash [Homebrew]
brew install HasanKhatib/tap/hikma
```

```bash [Script]
curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash
```

:::

Both give you the `hikma` command. The script installs to `~/.local/bin`; on Windows, run it in Git Bash.

::: details Install options and verifying the download
- The script downloads the matching release and checks it against `checksums.txt`. When the GitHub CLI (`gh`) is installed and signed in, it also verifies the release's build attestation, which proves the archive was built by this repository's release workflow, and it tells you when it skips that check.
- To verify an archive yourself: `gh attestation verify hikma_darwin_arm64.tar.gz -R HasanKhatib/hikma-ai`.
- Set `HIKMA_INSTALL_DIR` to change where it installs, or pass a version: `... | bash -s -- v1.0.0`. On Windows make sure the install directory is on your `PATH`.
- Set `HIKMA_REQUIRE_ATTESTATION=1` to make the script fail instead of skipping the attestation check when `gh` is missing.
- Installed the old Homebrew cask? Switch once: `brew uninstall --cask hikma`, then `brew install HasanKhatib/tap/hikma`.
- To build from source: `go install github.com/hasankhatib/hikma-ai/cmd/hikma@latest`.
:::

To upgrade later, run [`hikma upgrade`](/commands/upgrade).

Check the install:

```bash
hikma --version
hikma doctor
```

`doctor` needs `git`. The GitHub CLI (`gh`) is optional and only used by [`skill push`](/commands/skill-push) and for private GitHub repositories.

## 2. Pick your path

| I want to… | Go to |
|---|---|
| Use skills from a repository | [Use skills from any repository](#use-skills-from-any-repository) |
| Publish my own skills | [Run your own registry](#run-your-own-registry) |
| Set up a repository for my team | [Set up a team repository](#set-up-a-team-repository) |
| Be guided | Run `hikma` with no arguments for the [menu](/guides/menu) |

### Use skills from any repository

No setup is needed to install from a repository you name:

```bash
hikma skill install owner/repo            # pick from the skills in that repo
hikma skill install owner/repo my-skill   # install one by name
hikma skill install owner/repo my-skill --ref v1.0   # pin a tag, branch, or commit
```

Skills land in your agent's folder (`.agents/skills` by default, `.claude/skills` for Claude) and are recorded in `.hikma/lock.json`. See [skill install](/commands/skill-install).

### Run your own registry

Set a registry once and use bare names:

```bash
hikma config set registry owner/repo
hikma skill list
hikma skill install my-skill
```

Then create and publish a skill:

```bash
hikma skill create my-skill --description "What it helps with"
# edit .claude/skills/my-skill/SKILL.md (or .agents/skills/my-skill/SKILL.md)
hikma skill push my-skill
```

`push` validates the skill, shows the target registry and asks you to confirm, then commits the skill to your registry (or opens a pull request from your fork if you lack write access, or with `--pr`). See [skill push](/commands/skill-push).

### Set up a team repository

To prepare a repository for one or more agents:

```bash
hikma init --agent claude,codex --registry owner/repo --skill my-skill
```

This writes `AGENTS.md`, saves the agents and registry in `.hikma/config.json`, and installs the skill into `.claude/skills` and `.agents/skills`. Commit `.hikma/` so teammates share the setup. See [init](/commands/init) and [Set up a team repo](/commands/init#share-it-with-your-team).

## Next

- [The menu](/guides/menu): the guided way to do all of this
- [Command reference](/commands/)
- [Skill format](/guides/skill-format)
- [Run a registry](/guides/registry)
- [Troubleshooting](/guides/troubleshooting)
