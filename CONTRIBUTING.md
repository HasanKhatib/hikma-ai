# Contributing

Thanks for improving Hikma AI.

## Local checks

Run these before opening a pull request (CI runs them on Linux, macOS, and Windows):

```bash
gofmt -w cmd internal
go vet ./...
go test -short ./...                     # the main suite; CI runs it on every change
go test ./e2e                            # after any change to the menu, prompts, or color: drives the real binary in a pty
go mod tidy          # go.mod and go.sum must not change
bash scripts/test-install.sh   # if you touched scripts/
npx agentkan validate docs/board   # if you touched docs/board
(cd docs/site && npm ci && npm run docs:build)   # if you touched docs/site
```

## Tasks

[Task](https://taskfile.dev) wraps the common commands. Run `task` to list them.

```bash
task ci               # everything CI runs: lint, tidy check, tests, build, board, docs
task test             # Go tests, without the terminal tests
task test:terminal    # the pty tests that drive the real menu and color
task docs:e2e         # docs site in real browsers and phone profiles; manual, not part of CI (run the Docs e2e workflow from the Actions tab)
task run -- skill list   # build and run hikma with arguments
task docs:dev         # docs with live reload at http://localhost:5173/hikma-ai/
task docs:preview     # production build served like GitHub Pages at http://localhost:4180/hikma-ai/
task docs:pages       # same, from a clean npm install
task docs:stop        # stop a docs server these tasks started
```

## Registry boundary

Hikma AI is the CLI repository only. Do not add a top-level `skills/` registry folder here. Tests build their registries on the fly in temporary directories; do not commit registry content.

## Tests

- Command tests run the CLI in-process against a temporary git repository; see `cmd/hikma/commands/skill_install_test.go`.
- Calls to the `gh` CLI go through one function (`ghRun`) and are tested with a fake GitHub; see `skill_push_test.go`.
- `e2e/` runs the real binary in a pseudo-terminal. Keep it to what needs a real terminal (the menu draws and exits, color follows the terminal); anything a fake `Prompter` can cover belongs in a command test. CI runs it in its own `Terminal` workflow, apart from `go test -short`, and the release hook skips it.
- `e2e/github_test.go` runs create, push, install, verify, update against a real throwaway registry. It is skipped unless `HIKMA_E2E_REPO=owner/repo` and `GH_TOKEN` are set; the manual `GitHub e2e` workflow runs it on three OSes once the `E2E_REGISTRY` variable and `E2E_GITHUB_TOKEN` secret exist (the workflow file explains the setup).
- `TestGhSkillInterop` runs the real `gh skill install` and is skipped unless `HIKMA_GH_CANARY=1`; the `Canary` workflow runs it weekly against the newest gh, because `gh skill` is a preview and may change.
- Config tests isolate `HOME`, `XDG_CONFIG_HOME`, and `APPDATA` so they never touch your real configuration.

## Pull requests

- Keep changes focused and open them against `main`.
- Include tests for behavior changes.
- Update the README and the docs site (`docs/site`, one page per command) when commands, config, or registry behavior changes.
- Use clear commit messages; conventional prefixes (`feat:`, `fix:`, `docs:`, `ci:`) are welcome.

## Planning

Roadmap and decisions live in `docs/board/`. See `AGENTS.md` for the board rules and the project's non-goals.

## Releasing

Releases are tag-based. Pushing a `v*` tag runs GoReleaser, publishes binaries to GitHub Releases, attests the archives and `checksums.txt`, and writes the Homebrew formula (`scripts/render-formula.sh`) into `HasanKhatib/homebrew-tap`, using the `HOMEBREW_TAP_DEPLOY_KEY` repository secret (a write-enabled deploy key on the tap). Pre-release tags such as `v0.3.0-rc.1` build and attest but do not update the tap.

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

Try a release locally with `goreleaser release --snapshot --clean`.
