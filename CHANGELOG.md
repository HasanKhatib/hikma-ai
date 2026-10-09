# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims to follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed
- `hikma update` and `hikma --help` point to `brew upgrade` and the install script instead of stale text; examples use placeholders instead of a specific registry.
- `hikma skill install` prints a file count per folder instead of every file, and `skill remove` deletes agent folders it leaves empty.
- File hashes in `.hikma/lock.json` ignore CRLF versus LF line endings in text files, so a lockfile written on one platform verifies on another. Binary files are still hashed byte for byte.

### Added
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

[Unreleased]: https://github.com/HasanKhatib/hikma-ai/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/HasanKhatib/hikma-ai/releases/tag/v0.1.0
