# Hikma AI Migration Plan

## Product Boundary

Hikma AI is the open-source CLI only. It should not contain its own bundled skill registry folder. Registries are separate repositories or local paths chosen by users.

The planned first personal registry is:

```text
hasankhatib/ai
```

This registry can be used in examples, but it should not be hardcoded as the only supported registry. If a default registry is provided later, it must be an explicit product decision and easy to override.

## Target User Flow

The first release should support this flow:

```bash
hikma init
hikma registry set hasankhatib/ai
hikma skill list
hikma skill install agentkan
hikma skill create my-skill
```

Optional flags can make this single-shot:

```bash
hikma init --agent claude --registry hasankhatib/ai
```

## What Can Migrate As-Is

These areas can likely be copied first, then renamed and tested:

- Cobra command structure.
- Go package layout for commands, scaffold, skills, registry access, and preflight checks.
- Embedded template mechanism using `go:embed`.
- Skill create/install/update command concepts.
- GitHub operations through the `gh` CLI.
- Existing command tests that do not depend on private names or fixed registry URLs.
- Release automation concepts such as GoReleaser, checksums, and GitHub Releases.

## What Must Change

These areas must change before the repo is public-release ready:

- Rename module path, binary name, command examples, package metadata, and docs to Hikma AI and `hikma`.
- Remove all organization-specific names, URLs, registry defaults, templates, and examples.
- Remove or omit any top-level `skills/` registry folder from this repo.
- Replace compile-time registry constants with configurable registry resolution.
- Make registry selection explicit through project config, user config, environment variables, or flags.
- Ensure generated scaffold output does not mention private infrastructure or any fixed registry owner.
- Decide whether GitHub authentication is required only for private registries and publishing, not for read-only public registry operations if anonymous clone works.
- Update release workflows to work from the public repository and personal GitHub account.
- Update install docs for public users who do not have private repository access.

## Registry Configuration Design

Recommended precedence:

1. Command flag: `--registry`.
2. Environment variable: `HIKMA_REGISTRY`.
3. Project config committed to the target repo, for example `.hikma/config.json`.
4. User config, for example `$HOME/.config/hikma/config.json`.
5. No default for v0.1 unless explicitly chosen.

Recommended accepted registry values:

- GitHub shorthand: `hasankhatib/ai`.
- HTTPS URL: `https://github.com/hasankhatib/ai.git`.
- SSH URL: `git@github.com:hasankhatib/ai.git`.
- Local path for development and tests.

The CLI should normalize these values internally but preserve the user-facing value in config where practical.

## Registry Shape

The separate registry repository should own the reusable content. A simple shape for `hasankhatib/ai` could be:

```text
skills/
  agentkan/
    SKILL.md
    assets/
    references/
    scripts/
templates/
  scaffold/
  skill/
registry.json
```

Hikma AI should validate the minimum shape it needs for each command. For example, `skill install` needs `skills/<name>/SKILL.md`, while `skill list` can use either `registry.json` or a directory scan.

## CLI Repository Shape

The Hikma AI repository should contain only source, tests, docs, and release configuration:

```text
cmd/hikma/
internal/
templates/
docs/
docs/board/
.github/workflows/
README.md
CLAUDE.md
go.mod
```

It should not contain:

```text
skills/
```

unless that folder is clearly test fixture data under a path like `internal/testdata/registry/skills/`.

## Migration Sequence

### Step 1: Clean Import

Import the source CLI into Hikma AI with a clean commit after the audit. Prefer a clean copy over full git history if the old history contains private context or irrelevant release automation.

### Step 2: Rename

Rename:

- Binary: `hikma`.
- Module path: the final public GitHub module path.
- Command references in tests and docs.
- Generated file comments and examples.
- Release asset names.

### Step 3: Registry Decoupling

Remove compile-time fixed registry behavior. Add config resolution and tests for precedence. Commands that touch GitHub should still use `gh` where needed, but public read-only install from a public registry should work without unnecessary authentication if the implementation supports it safely.

### Step 4: Template Neutralization

Rewrite scaffold templates so they generate neutral Claude-first files by default. Other agents can remain supported behind flags only if they are already implemented cleanly.

### Step 5: Tests And Fixtures

Create local test fixtures for registry behavior under `internal/testdata/`. Do not use the real `hasankhatib/ai` registry in unit tests. Integration tests can optionally use a public fixture registry later.

### Step 6: Documentation

Write docs for:

- Quickstart.
- Installation.
- Registry setup.
- Registry repository structure.
- Skill authoring.
- Skill install/update.
- Troubleshooting GitHub authentication.
- Using `hasankhatib/ai` as an example registry.

### Step 7: Website

Keep GitHub Pages useful for docs or redirects. Cloudflare Pages on `alkhatib.tech` can host the polished public site, while the repo can publish reference docs from `docs/`.

### Step 8: Release

Set up GoReleaser or equivalent release automation to publish binaries and checksums. Start with GitHub Releases if package managers add too much friction. Add Homebrew or other channels after the first stable release path is proven.

## Release Readiness Checklist

- `go test ./...` passes.
- `go vet ./...` passes if used.
- The binary builds as `hikma`.
- `hikma --help` has neutral copy and examples.
- `hikma init --agent claude --registry hasankhatib/ai` works or has documented equivalent steps.
- `hikma registry set hasankhatib/ai` persists config correctly.
- `hikma skill list` reads from the configured external registry.
- `hikma skill install <name>` installs into the selected agent path.
- The repository has no top-level `skills/` registry folder.
- No private names, URLs, tokens, or credentials appear in source, docs, tests, workflows, or generated outputs.
- README has install instructions, quickstart, and registry explanation.
- License is present.
- Release workflow publishes binaries and checksums from a tag.
- GitHub Pages or docs publishing is either configured or explicitly deferred.

## Open Decisions

- Confirm license.
- Confirm whether `hikma init` should ask for a registry interactively or require a follow-up `hikma registry set`.
- Confirm whether public read-only registry operations can bypass `gh auth` when using HTTPS clone.
- Confirm whether v0.1 supports only Claude scaffolding or multiple agent targets.
- Confirm whether GitHub Pages is the canonical docs site or a mirror of content published under `alkhatib.tech`.
