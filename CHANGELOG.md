# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims to follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- The menu asks for a description when you create a skill, links a new guide on writing and editing skills, and passes `--description` to `hikma skill create`. The guide, [Write and edit a skill](https://hasankhatib.github.io/hikma-ai/guides/write-a-skill), covers the description, `scripts/`, `references/`, `assets/`, and hooks, with links to Anthropic's and OpenAI's published guidance.
- The publish menu marks the skills that already exist in your registry with `[exists]`.
- `hikma skill list` and the `hikma skill install` picker mark skills you already have with `[installed]`.
- Lists where you tick options show a checkbox (`[ ]`, `[x]`).

### Fixed
- `hikma skill create --description` (and the owner and team) no longer breaks the `SKILL.md` when the text contains a quote or a backslash.
- A question that may be left empty in the menu, such as the optional registry in project setup, can now be left empty in a terminal; the prompt used to refuse an empty answer.
- `hikma completion` no longer stops at "already set up". It says the line only works in windows opened after it was added and how to load it now, and for zsh it checks that `compinit` comes before the hikma line, since otherwise zsh prints `command not found: compdef`.

### Changed
- The install menu lists the skills you already have last, crossed out and marked `[installed]`, instead of hiding them; ticking one does nothing. When every skill in the registry is already installed it now says so and lists them, instead of showing an `error:`.

## [1.2.0] - 2026-10-20

### Added
- `hikma skill import <source> [<name>]` brings a skill you found into your registry. It reads `owner/repo`, an exact folder (`owner/repo/skills/name`), a GitHub `/tree/` or `/blob/` link, or a local folder. It shows a review first (files, licence, scripts, executables, binaries, links outside the folder), asks, and publishes it the way `skill push` does: committed to the registry's default branch (whatever it is called) when you can write, as a pull request when a branch rule requires one or you cannot write (through your fork), and always as a pull request with `--pr`. Files are copied byte for byte from git objects, except that `metadata.github-*` is removed from `SKILL.md` so `gh skill update` cannot bypass your registry, and `--as` rewrites the name line. It refuses an existing skill (any letter case) without `--force`, takes at most 100 files and 5 MiB, and writes `skills/<name>/.hikma-origin.json` with the source, commit, tree, licence, file hashes, and every change from upstream. `--dry-run` shows the plan without contacting the registry, and `--json` prints it as a versioned document. See [skill import](https://hasankhatib.github.io/hikma-ai/commands/skill-import).
- `hikma skill install` takes `--skill <name>`, repeatable, to install several skills from one checkout of a source. Every name is checked before anything is written, one confirmation covers them all, a summary lists each skill as ok or failed, and the command exits non-zero naming the ones that failed. With a registry configured the source can be left out. See [skill install](https://hasankhatib.github.io/hikma-ai/commands/skill-install).

### Changed
- The `hikma` menu is now a wizard you stay in. It clears the terminal (keeping your scrollback; skipped under `TERM=dumb` and `ACCESSIBLE`), runs a task, shows the result, and asks whether to go back to the menu or quit. Ctrl-C inside a task, including while a registry is being read or a command runs, goes back to the menu; Ctrl-C at the menu leaves. A failed task shows its error and returns to the menu, and the equivalent command is printed only after a task that worked. A line under the banner shows the registry, the agent, and how many skills are installed here.
- The menu's setup entries (project setup, agent, registry, completion, setup check) moved under "Setup and settings".
- "Install a skill from your registry" is now "Install skills from your registry", and it works end to end. Before, it ran `hikma skill install` with no skill name and failed with "missing argument". Now it asks for the registry and agent when unset, reads the registry with a spinner, lists the skills you do not have yet (filter with `/`, `ctrl+a` ticks all), shows a plan, asks once, and runs one command. Failed skills can be retried from "What next?".
- Long menu lists are held to the terminal height (never fewer than four rows) and can be filtered by typing `/`; each menu shows its keys. The cursor stays on the entry you used last. Esc backs out of a prompt like Ctrl-C (on a filterable list Esc clears the filter), and Ctrl-C or Esc at "What next?" goes back to the menu instead of quitting.
- Ctrl-C now stops work that has no child process too: the menu passes a cancellable context to the command and to the `git` or `gh` clone behind it, `skill install --skill` stops between skills, and a second Ctrl-C ends hikma. Cancelling the registry spinner stops the clone.
- The install flow keeps your agent when you mistype the registry and asks for another one, shows a message when you decline the plan, lists skills already installed for your agent only (a copy for another agent no longer hides it), and shows descriptions in the plan. When some skills of an install fail, the menu prints the command that ran and the command that retries only the ones that failed, and the error line no longer repeats the summary.
- Accessible mode (`ACCESSIBLE` set) draws the banner once and asks for lists by number.
- `hikma init` and `hikma doctor` say how publishing to a GitHub registry will go: a commit to the default branch, a pull request because the branch requires one, or a pull request through your fork. Nothing is asked or saved, because it depends on who you are and on branch rules; with no `gh`, no sign-in, a `--dry-run`, or a registry outside GitHub they print nothing.
- `hikma skill install` now takes a folder in a GitHub repository (`owner/repo/skills/name`) and GitHub `/tree/<branch>/<folder>` links, as well as `owner/repo`. Credentials in a pasted URL are never printed.
- `hikma skill push` retries when GitHub rate limits it (HTTP 429 or a secondary limit): after a minute, then two, then four. If it still fails, the error says how many files were uploaded and that nothing was committed.

## [1.1.0] - 2026-10-19

### Changed
- `hikma upgrade` now upgrades hikma instead of printing instructions. It shows the plan (`brew upgrade` for a Homebrew install, otherwise the install script, which checks the checksum and attestation), asks first, and replaces the binary where it is. `--yes` skips the question; without a terminal it prints the plan and exits non-zero. The old `hikma update` command is now `hikma upgrade`.

## [1.0.0] - 2026-10-18

The first stable release. From here, commands, flags, exit codes, JSON documents, and the lockfile and config schemas follow the [versioning policy](https://hasankhatib.github.io/hikma-ai/reference/policy).

### Added
- A Reference section in the docs: exit codes and JSON, configuration and environment, flags and terminal behavior, and the versioning and deprecation policy.
- `--json` on `skill verify`, `skill diff`, and `sync`. Each prints one versioned document on stdout, with progress on stderr. `sync --json` never prompts, so it needs `--yes` or `--dry-run`.
- Exit codes are now a documented contract: `0` success, `1` failure, `2` wrong usage (unknown command or flag, missing argument), `3` installed skills differ from the lockfile (`skill verify`, `sync --frozen`). They were `0` and `1` only. A new reference page lists them and every JSON document.

### Changed
- The install warning for an agent that reads another agent's folder now says it "may use either copy" instead of "may list each skill twice": Copilot and OpenCode list one copy.
- Errors now print once, as `error: <message>`, and say what to do next. Before, some failures (already installed, already exists, skill not found locally) printed their message and then a second short line. Mistakes in how flags combine (`--all` with a name, `--global` without `--installed`) exit `2`. "Not installed" errors name the lockfile for the scope you used and point to `hikma skill list --installed`.
- Every `--json` document is now an object with a `version` field instead of a bare array or object: `skill list` is `{ "version": 1, "skills": [...] }`, `skill info` is `{ "version": 1, "skill": {...} }`, `skill list --installed` is `{ "version": 1, "skills": [...] }`, and `config list` is `{ "version": 1, "entries": [...] }`. Scripts that read the old arrays need a one-line change.
- A lockfile or config file written by a newer `hikma` is now refused with an error that says to upgrade, and is never rewritten. Before, an older `hikma` silently rewrote a newer lockfile. Config files now carry a `version` field, added the next time `hikma` writes them; a file without one is read as version 1.

## [0.5.0] - 2026-10-18

### Added
- `hikma` with no arguments in a terminal opens a menu for the common tasks (install, browse, update, verify, create, publish, set up a project, choose an agent, set a registry, install completion, check your setup). It runs the same command a script would and prints the equivalent `hikma` command afterwards. Without a terminal, `hikma` still prints help, so scripts and CI never wait for input. The menu asks for what a command needs (the registry and agent for install, a skill picker for publish and update, the agents for project setup), so the command it prints is complete.
- A banner with the logo's frame and prompt shows above the menu, colored in a terminal. Output stays plain when piped, under `NO_COLOR`, or with `TERM=dumb`.
- `hikma completion` with no shell detects yours, shows the file it would change, and asks before installing. `--install` does the same for a named shell; `--yes` skips the question. `hikma completion <shell>` still only prints the script.

- Global `--no-input` and `--no-color` flags. `--no-input` stops every prompt, even in a terminal: a command that would have asked fails and names the flag to pass (usually `--yes`), and bare `hikma` prints help instead of the menu. `--no-color` does the same as `NO_COLOR`.

- Prompts (menu, confirmations, pickers) now use Charm's `huh` instead of the archived `survey`. They look slightly different, honor `--no-color`, `NO_COLOR`, and `--no-input`, and fall back to plain line prompts under `TERM=dumb` or when `ACCESSIBLE` is set.

### Fixed
- `hikma completion` writes to the command's output, and the zsh instructions no longer tell you to write to `${fpath[1]}`, which is often a folder you cannot write to or one zsh never reads. Use `source <(hikma completion zsh)` in `~/.zshrc`.
- The Homebrew formula installs shell completions.
- Bash completion installs into `~/.bashrc` on Linux and Windows even when that file does not exist yet (it used to fall back to `~/.bash_profile`, which interactive shells there never read). macOS still uses `~/.bash_profile`.

### Changed
- The install warning for an agent that reads another agent's folder now says it "may use either copy" instead of "may list each skill twice": Copilot and OpenCode list one copy.
- `hikma skill push` commits straight to the registry's default branch when you have write access, instead of opening a pull request. Use `--pr` for the pull request flow. Without write access it still forks and opens a pull request. If the default branch is protected, the error suggests `--pr`. The confirmation states which of the two will happen, and `--codeowners` requires `--pr`. This changes the default for anyone with write access. When the default branch has a rule that requires pull requests, `skill push` opens one on its own and says why, instead of failing the commit.
- `hikma doctor`, `skill verify`, and `skill list --installed` use color in a terminal: `ok` green, `FAIL` red, `warn` yellow, fix hints and table headings dim, and the status column colored (modified yellow, missing red, untracked dim). Output stays plain when piped or under `NO_COLOR`.
- Warnings are yellow and `FAIL`/errors red in a terminal. `NO_COLOR` turns it off, and piped output stays plain.
- `hikma skill push` reports unfilled template placeholders as warnings and continues to its confirmation, so a draft can be pushed for review. `registry validate` still fails them.
- `hikma skill adopt` on a skill with no `gh skill` metadata explains why and shows the commands to run, using your configured registry when there is one.
- Mistyped arguments, commands, and flags now say what is wrong and what to type: the usage line, a "did you mean" suggestion, and the right help command. `hikma completion zshs` names the supported shells instead of "invalid argument".

## [0.4.0] - 2026-10-12

### Added
- `--global` (`-g`) on `skill install`, `update`, `remove`, `adopt`, `verify`, `diff`, `list --installed`, and `sync` works in your home directory with its own lockfile at `~/.hikma/lock.json`. A global write prints the absolute paths and asks first (`--yes` skips it), and hikma refuses to write through a symlinked folder. `init` stays project-only.
- A warning, never a block, when a skill name is installed at both project and global scope: on install, in `list --installed` (which now has a scope column), in `sync`, and in `doctor`.

### Changed
- The install warning for an agent that reads another agent's folder now says it "may use either copy" instead of "may list each skill twice": Copilot and OpenCode list one copy.
- `skill update` now rejects lockfile entries whose path is not a clean `<agent skills folder>/<name>`, as `sync` already did.

## [0.3.0] - 2026-10-12

### Changed
- The install warning for an agent that reads another agent's folder now says it "may use either copy" instead of "may list each skill twice": Copilot and OpenCode list one copy.
- Homebrew now installs a formula instead of a cask. Run `brew uninstall --cask hikma` once if you installed the cask, then `brew install HasanKhatib/tap/hikma`.
- `hikma skill update --force` now restores a skill you edited even when upstream has not changed; before, it reported "up to date" and kept your edits.
- `hikma skill update` asks before applying an update that changes `scripts/`. Without a terminal it refuses until you pass `--yes`.

### Added
- Release archives and `checksums.txt` carry GitHub build attestations. `scripts/install.sh` verifies them with `gh` for v0.3.0 and later (it says when it skips, and `HIKMA_REQUIRE_ATTESTATION=1` makes a skip a failure).
- Homebrew is published as a formula written by the release workflow, so installs need no quarantine workaround and the Homebrew deprecation warning is gone.
- Custom agents: define an agent under `customAgents` in config (`skillsDir`, optional `instructionFile`) and use it with `init`, `skill install`, `update`, and `sync`. Paths are validated when the config loads.
- `hikma config add-agent <name>` defines a custom agent with y/n prompts that show the config file, or with flags for scripts; `hikma config remove-agent <name>` removes it and its references. `hikma config list` shows custom agents.
- A warning when an install writes a skill to two folders that one selected agent reads (Copilot or OpenCode together with Claude). The install proceeds.
- `skill install --force` and `sync --force` refuse to replace an existing folder that is not a skill (no `SKILL.md`).
- `hikma skill verify [<name>]` re-hashes installed skills against `.hikma/lock.json`, offline, and exits non-zero on any difference.
- `hikma sync --frozen` fails, changing nothing, when the lockfile cannot be reproduced exactly: local changes, an entry without a pinned commit, or a source that cannot be fetched. For CI.
- `hikma skill diff <name>` shows local edits and the upstream changes an update would bring, marking files changed on both sides; `--patch` prints the contents. It changes nothing.

## [0.2.0] - 2026-10-10

### Changed
- The install warning for an agent that reads another agent's folder now says it "may use either copy" instead of "may list each skill twice": Copilot and OpenCode list one copy.
- `hikma init --agent claude` writes only `AGENTS.md`; Claude Code reads it when there is no `CLAUDE.md`. The generated `CLAUDE.md` stub is gone.
- `skill create` generates a SKILL.md that follows the Agent Skills spec: `metadata` holds only string `owner` and `team`, and the placeholder `compatibility`, `email`, `policy`, and `last_validated` fields are removed.
- `registry validate` checks the Agent Skills spec: name and description length, `compatibility` length, string-only `metadata`, and warns on fields outside the spec, non-spec names, and SKILL.md files over 500 lines.
- Docs: new "Agents and folders" guide, a custom site theme with a one-screen homepage and a new logo; the use-cases section is removed.
- `hikma update` and `hikma --help` point to `brew upgrade` and the install script instead of stale text; examples use placeholders instead of a specific registry.
- `hikma skill install` prints a file count per folder instead of every file, and `skill remove` deletes agent folders it leaves empty.
- File hashes in `.hikma/lock.json` ignore CRLF versus LF line endings in text files, so a lockfile written on one platform verifies on another. Binary files are still hashed byte for byte.

### Added
- `hikma skill install --pin <ref>` is an alias of `--ref`, matching `gh skill`.
- `hikma skill adopt <name>` records a skill installed by `gh skill` (or any skill, with `--source`) in `.hikma/lock.json`. `skill list --installed`, `doctor`, and `skill update` now point at skills installed by `gh skill` that Hikma does not track.
- `hikma registry validate --format github` prints GitHub workflow annotations with file and line, and the repository is now a reusable GitHub Action (`uses: HasanKhatib/hikma-ai@<tag>`) that validates a registry on pull requests.
- `hikma skill remove <name>` deletes an installed skill and its lockfile entry, keeping untracked skills and local edits unless `--force`.
- `hikma skill list --installed` shows what is installed in this repository (`ok`, `modified`, `missing`, or `untracked`).
- `hikma sync` restores every skill in `.hikma/lock.json` at its recorded commit, verifies the files against the recorded hashes, protects local edits, and rejects lockfile entries outside the agent skill folders.

## [0.1.0] - 2026-10-09

First public release.

### Added
- Install skills from any source: GitHub `owner/repo`, a git URL, or a local path, with `--ref` to pin a version; pick from a list when no skill name is given.
- `.hikma/lock.json` records each install's source, commit, and file hashes. `hikma skill update` uses the recorded source, blocks on local edits, and flags changes to `scripts/`.
- Layered configuration (flag, environment, project `.hikma/config.json`, user config) with `hikma config list|get|set|unset|path`, showing where each value comes from.
- `hikma init` sets up one or more agents (Claude, Codex, Copilot, OpenCode): instruction files, shared project config, and skills in each agent's folder.
- `hikma skill push` confirms the target registry, works without write access through a fork, and never force-pushes.
- `hikma registry validate` checks skills against the skill format and your naming setting (`loose` or `kebab-case`).
- `hikma doctor`; shell completions.
- Homebrew cask and an install script for Windows (Git Bash) and other platforms, with checksum verification.

[Unreleased]: https://github.com/HasanKhatib/hikma-ai/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/HasanKhatib/hikma-ai/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/HasanKhatib/hikma-ai/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/HasanKhatib/hikma-ai/compare/v0.5.0...v1.0.0
[0.5.0]: https://github.com/HasanKhatib/hikma-ai/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/HasanKhatib/hikma-ai/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/HasanKhatib/hikma-ai/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/HasanKhatib/hikma-ai/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/HasanKhatib/hikma-ai/releases/tag/v0.1.0
