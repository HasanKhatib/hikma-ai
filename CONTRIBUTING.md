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
task docs:dev         # docs with live reload at http://localhost:5173/
task docs:preview     # production build served like GitHub Pages at http://localhost:4180/
task docs:pages       # same, from a clean npm install
task docs:stop        # stop a docs server these tasks started
```

## What CI runs

`hygiene` (commit and title rules, gofmt, `go mod tidy`) and `vulncheck` run on every change. The other jobs run only when the change can affect them, decided by `scripts/ci-changes.sh` from the changed files:

| Job | Runs when these change |
|---|---|
| `test` (Linux, macOS, Windows) | `cmd/`, `internal/`, `e2e/`, `go.mod`, `go.sum`, `.goreleaser.yml` |
| `vet` (one Linux runner per system, set with `GOOS`) | the same |
| `release-dry-run` (pull requests only) | the same, plus `scripts/` and `release.yml` |
| `install-script` | `scripts/`, `.goreleaser.yml` |
| `docs-build` | `docs/site/`, `pages.yml` |

A change to `ci.yml` or `scripts/ci-changes.sh` runs everything, and so does a change CI cannot list (no base commit, or more than 300 files). The pty tests have their own path filter in `terminal.yml`. Skipping a job never skips the check you should run yourself: the commands above are the full list, and `task ci` runs them all.

## Docs hosting

The docs site is published at **https://hikma.alkhatib.tech/**. It is built from `docs/site` and deployed by `.github/workflows/pages.yml` to GitHub Pages whenever `docs/site/**` changes on `main`.

- The address comes from a custom domain on GitHub Pages. `docs/site/public/CNAME` holds it, so keep that file. The site is built with base `/`; set `VITEPRESS_BASE` only to serve it from a sub-path.
- DNS is a `hikma` CNAME to `hasankhatib.github.io`, set to DNS only (not proxied) so GitHub can issue the HTTPS certificate. It lives in the owner's Cloudflare account.
- The old `hasankhatib.github.io/hikma-ai/` address redirects to the new one.

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

Releases are automatic. Nobody tags by hand and there is no changelog file to edit.

Merging a pull request to `main` can release. The **Release** workflow (`.github/workflows/release.yml`) runs on every push to `main`:

1. **Wait for checks.** It waits until every other workflow on that commit has finished and passed, so a failing check (including the pty tests) blocks a release.
2. **Semantic release.** [semantic-release](https://semantic-release.gitbook.io) reads the conventional commits since the last tag (rules in `.releaserc.json`), picks the version, pushes the tag, and creates the GitHub release with generated notes. If no commit calls for a release, nothing happens and the rest is skipped.
3. **GoReleaser** builds the binaries at that tag and attaches them to the release. The archives and `checksums.txt` are attested, and the Homebrew formula (`scripts/render-formula.sh`) is written into `HasanKhatib/homebrew-tap` using the `HOMEBREW_TAP_DEPLOY_KEY` repository secret (a write-enabled deploy key on the tap).
4. **Verify** installs the published release on each OS the way users do.

The version comes from the commit titles, and a squash merge uses the pull request title, so write the title for the person who will read the release note:

| Title | Version |
|---|---|
| `fix:`, `perf:` | patch (1.3.0 to 1.3.1) |
| `feat:` | minor (1.3.0 to 1.4.0) |
| `feat!:`, or a `BREAKING CHANGE:` footer | major |
| `docs:`, `chore:`, `ci:`, `test:`, `refactor:`, `style:`, `build:` | no release |

CI rejects a pull request title that does not start with one of these types.

If **Build** fails after the tag exists, re-run the failed jobs of that run: semantic-release will not release the same commits twice. Do not push `v*` tags by hand.

Try a release locally with `goreleaser release --snapshot --clean`.
