# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims to follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- `hikma completion` with no shell detects yours, shows the file it would change, and asks before installing. `--install` does the same for a named shell; `--yes` skips the question. `hikma completion <shell>` still only prints the script.

### Fixed
- `hikma completion` writes to the command's output, and the zsh instructions no longer tell you to write to `${fpath[1]}`, which is often a folder you cannot write to or one zsh never reads. Use `source <(hikma completion zsh)` in `~/.zshrc`.
- The Homebrew formula installs shell completions.

### Changed
- `hikma skill push` commits straight to the registry's default branch when you have write access, instead of opening a pull request. Use `--pr` for the pull request flow. Without write access it still forks and opens a pull request. If the default branch is protected, the error suggests `--pr`. The confirmation states which of the two will happen, and `--codeowners` requires `--pr`. This changes the default for anyone with write access.
- Warnings are yellow and `FAIL`/errors red in a terminal. `NO_COLOR` turns it off, and piped output stays plain.
- `hikma skill push` reports unfilled template placeholders as warnings and continues to its confirmation, so a draft can be pushed for review. `registry validate` still fails them.
- `hikma skill adopt` on a skill with no `gh skill` metadata explains why and shows the commands to run, using your configured registry when there is one.
- Mistyped arguments, commands, and flags now say what is wrong and what to type: the usage line, a "did you mean" suggestion, and the right help command. `hikma completion zshs` names the supported shells instead of "invalid argument".

## [0.4.0] - 2026-10-12

### Added
- `--global` (`-g`) on `skill install`, `update`, `remove`, `adopt`, `verify`, `diff`, `list --installed`, and `sync` works in your home directory with its own lockfile at `~/.hikma/lock.json`. A global write prints the absolute paths and asks first (`--yes` skips it), and hikma refuses to write through a symlinked folder. `init` stays project-only.
- A warning, never a block, when a skill name is installed at both project and global scope: on install, in `list --installed` (which now has a scope column), in `sync`, and in `doctor`.

### Changed
- `skill update` now rejects lockfile entries whose path is not a clean `<agent skills folder>/<name>`, as `sync` already did.

## [0.3.0] - 2026-10-12

### Changed
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

[Unreleased]: https://github.com/HasanKhatib/hikma-ai/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/HasanKhatib/hikma-ai/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/HasanKhatib/hikma-ai/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/HasanKhatib/hikma-ai/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/HasanKhatib/hikma-ai/releases/tag/v0.1.0
