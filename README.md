<p align="center"><img src="docs/site/public/logo/Logo.png" alt="Hikma" width="140"></p>

# Hikma AI

**Skills follow you.** Hikma AI is a CLI for owning your AI-agent skills: publish them from any project to a registry repo you control, and install them into Claude, Codex, Copilot, OpenCode, or a custom agent layout.

This repository is the CLI only. Registries are plain repos you configure; none is bundled.

## Use Hikma

Read the documentation at **[hasankhatib.github.io/hikma-ai](https://hasankhatib.github.io/hikma-ai/)**. It has a page for every command.

Install, then check your setup:

```bash
brew install HasanKhatib/tap/hikma     # macOS and Linux
hikma doctor
```

On Windows or without Homebrew, use the install script from [Getting started](https://hasankhatib.github.io/hikma-ai/getting-started).

A first session:

```bash
hikma skill install anthropics/skills pdf        # from any repo, no setup
hikma config set registry owner/registry         # your own registry
hikma skill create my-skill --description "What it helps with"
hikma skill push my-skill                        # shows the target, asks to confirm
hikma skill import owner/repo some-skill         # bring a skill you found into your registry, after a review
hikma init --agent claude,codex --skill my-skill # set up a whole repo for your agents
```

Run `hikma` with no arguments in a terminal for a menu of the common tasks.

Guides: [agents and folders](https://hasankhatib.github.io/hikma-ai/guides/agents), [skill format](https://hasankhatib.github.io/hikma-ai/guides/skill-format), and [running a registry](https://hasankhatib.github.io/hikma-ai/guides/registry).

## Develop Hikma

Install [Go](https://go.dev/) and [Node.js](https://nodejs.org/). The GitHub CLI is only needed for `skill push`.

```bash
go build ./...
go test ./...
go vet ./...
npx agentkan validate docs/board
```

CI runs the checks in [CONTRIBUTING.md](CONTRIBUTING.md) on Linux, macOS, and Windows.

## Repository layout

- `cmd/hikma/commands/` holds one file per command, with its tests beside it.
- `internal/` holds the shared packages: `source` (resolve and check skills), `lock`, `config` (layers and the agent table), `scaffold`, and `ui`.
- `docs/site/` is the VitePress documentation site, one page per command. `docs/board/` is the roadmap.
- `scripts/` holds the install script and release helpers.
- `.github/workflows/` holds CI, docs, and release.

## How Hikma works

- A skill source is a GitHub `owner/repo`, any git URL, or a local path. Reading skills needs only `git`.
- Scripts and CI can rely on documented exit codes (`3` means installed skills differ from the lockfile) and versioned `--json` output: see [Exit codes and JSON](https://hasankhatib.github.io/hikma-ai/reference/output).
- Installs are recorded in `.hikma/lock.json` with the source, commit, and file hashes. `hikma sync` restores them on a fresh clone; `hikma skill update` stops on local edits.
- `hikma skill push` always shows the target registry and asks before writing. Nothing is force-pushed.
- `hikma skill import` copies a skill you found into your registry after a review of its files and licence, publishes it the way `push` does (a commit to the default branch when you can write, a pull request when the branch requires one or you cannot write), and records where it came from in `.hikma-origin.json`. It never overwrites an existing skill without `--force`. `init` and `doctor` tell you which way publishing will go.
- Agent layouts are data in one table, so adding an agent never touches command code.

## Release

Pushing a `v*` tag runs GoReleaser, publishes binaries and attestations to GitHub Releases, and updates the Homebrew formula. See [CONTRIBUTING.md](CONTRIBUTING.md#releasing) for the steps.

## Contributing

Open focused pull requests against `main` and run the local checks first. Use conventional commit prefixes (`feat:`, `fix:`, `docs:`, `chore:`, `ci:`). See [CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md).

## License

MIT. See [LICENSE](LICENSE).
