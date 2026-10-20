# Getting started

This page takes you from nothing to a project with your agent set up and a skill installed from your registry.

## 1. Install

::: code-group

```bash [Homebrew]
brew install HasanKhatib/tap/hikma
```

```bash [Script]
curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash
```

:::

The script downloads the matching release, checks it against `checksums.txt`, and installs to `~/.local/bin`. From v0.3.0 it also verifies the release's build attestation when the GitHub CLI (`gh`) is installed and signed in, and tells you when it skips that check. To verify an archive yourself: `gh attestation verify hikma_darwin_arm64.tar.gz -R HasanKhatib/hikma-ai`. Set `HIKMA_INSTALL_DIR` to change that, or pass a version: `... | bash -s -- v0.1.0`. On Windows run it in Git Bash and make sure the install directory is on your `PATH`. Set `HIKMA_REQUIRE_ATTESTATION=1` to make the script fail instead of skipping the attestation check when `gh` is missing; releases before v0.3.0 have no attestation and are checked against `checksums.txt` only.

Installed the old Homebrew cask? Switch once: `brew uninstall --cask hikma`, then `brew install HasanKhatib/tap/hikma`. To build from source instead: `go install github.com/hasankhatib/hikma-ai/cmd/hikma@latest`.

Check the install:

```bash
hikma --version
hikma doctor
```

`doctor` needs `git`. The GitHub CLI (`gh`) is optional and only used by [`skill push`](/commands/skill-push) and for private GitHub repositories.

## 2. Install a skill from any repository

No setup is needed to install from a repository you name:

```bash
hikma skill install owner/repo            # pick from the skills in that repo
hikma skill install owner/repo my-skill   # install one by name
hikma skill install owner/repo my-skill --ref v1.0   # pin a tag, branch, or commit
```

Skills land in your agent's folder (`.agents/skills` by default, `.claude/skills` for Claude) and are recorded in `.hikma/lock.json`. See [skill install](/commands/skill-install).

## 3. Set up a whole repository

To prepare a repository for one or more agents:

```bash
hikma init --agent claude,codex --registry owner/repo --skill my-skill
```

This writes `AGENTS.md`, saves the agents and registry in `.hikma/config.json`, and installs the skill into `.claude/skills` and `.agents/skills`. Commit `.hikma/` so teammates share the setup. See [init](/commands/init) and [Set up a team repo](/commands/init#share-it-with-your-team).

## 4. Use your own registry

Set a registry once and use bare names:

```bash
hikma config set registry owner/repo
hikma skill list
hikma skill install my-skill
```

## 5. Create and publish a skill

```bash
hikma skill create my-skill --description "What it helps with"
# edit .claude/skills/my-skill/SKILL.md (or .agents/skills/my-skill/SKILL.md)
hikma skill push my-skill
```

`push` validates the skill, shows the target registry and asks you to confirm, then commits the skill to your registry (or opens a pull request from your fork if you lack write access, or with `--pr`). See [skill push](/commands/skill-push).

## The menu

Run `hikma` with no arguments in a terminal and it opens a menu:

```text
$ hikma
╭──────────╮
│ >_       │  hikma
│          │  skills follow you
╰──────────╯

registry: acme/skills | agent: claude | 3 installed here
Ctrl-C or Esc inside a task comes back here. At this menu they quit.

? What do you want to do?
> Install skills from your registry
  Browse skills in your registry
  Update installed skills
  Verify installed skills
  Create a new skill
  Publish a skill to your registry
  Setup and settings >
  Quit
```

Pick an entry and Hikma asks for anything it needs, then runs the same command you could type yourself. It asks only for what is missing: the registry when none is set, the agent when none is chosen, a skill from this project's agent folders when you publish or update, and the agents to set up for a new project. Agent, registry, project setup, completion, and the setup check live under "Setup and settings".

### Installing skills

Install reads the registry (with a spinner), lists the skills you do not have yet, and lets you tick as many as you want. Long lists can be filtered by typing `/`, and `ctrl+a` ticks them all. Skills already installed for your agent are named but not offered: reinstalling would need `--force` and replace local edits, so use Update installed skills for those. A copy installed for a different agent does not hide the skill. Before anything is written it shows a plan and asks once:

```text
Will install 2 skills from acme/skills
  foo  Lint a repo
  bar  Format docs
Into .claude/skills (agent: claude)
Install these? Yes / No
```

It then runs one `hikma skill install --skill=foo --skill=bar` and prints the result and that command, so you can run it yourself next time:

```text
Installed 2 of 2
  ok      foo
  ok      bar

Equivalent command:
  hikma skill install --skill=foo --skill=bar
```

If some skills fail, the others are still installed, the result names each one that failed, and "What next?" offers to retry just those.

### What the menu does and does not do

- It is a front end. Every entry runs an ordinary `hikma` command, so the confirmation questions before a push, an update, or a file write are the ones that command always asks.
- It clears the screen and stays open. After each task it shows the result and asks "What next?": go back to the menu or quit. Errors are shown there too, so a failed task never closes the menu.
- The equivalent command is printed after a task that worked, not after a failure or a cancel. When some skills of an install fail, it prints the command that ran and the command that retries only the ones that failed.
- Ctrl-C or Esc inside a task goes back to the menu without doing anything: at a question, while a registry is being read, or while a command runs (a second Ctrl-C while it is stopping ends hikma). The same goes for "What next?". At the menu itself they leave. On a long list Esc clears the filter instead, and Ctrl-C goes back.
- The cursor stays on the entry you used last.
- A mistyped registry asks for another one without asking for your agent again.
- Clearing keeps your scrollback. In accessible mode (`ACCESSIBLE` set) and under `TERM=dumb` the screen is not cleared, the banner is drawn once, and lists are answered by number.

### Scripts and CI

The menu opens only when both input and output are a terminal. When either is piped or redirected, `hikma` with no arguments prints help and exits, and nothing waits for input.

Pass `--no-input` to any command to rule out prompts even in a terminal. A command that would have asked fails instead and names the flag to pass, such as `--yes` to confirm. Bare `hikma --no-input` prints help instead of opening the menu.

### Color

The banner and the command output use color in a terminal. Set `NO_COLOR=1` or `TERM=dumb`, or pass `--no-color`, to turn it off. Piped output is always plain, and no message relies on color alone.

## Next

- [Command reference](/commands/)
- [Skill format](/guides/skill-format)
- [Run a registry](/guides/registry)
