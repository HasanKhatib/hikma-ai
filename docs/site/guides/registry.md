# Registry Guide

Hikma AI installs skills from any repository. A **registry** is the repository you choose as your default source and as the target of `hikma skill push`. The CLI repository itself contains no skills.

## Skill Layout

Hikma discovers skills without an index file. It checks, in order:

1. `skills/<name>/SKILL.md`
2. `<name>/SKILL.md` at the repository root
3. a single `SKILL.md` at the repository root

```text
skills/
  my-skill/
    SKILL.md
    scripts/
    references/
```

`SKILL.md` frontmatter supplies the description and optional `metadata.owner`.

## Install From Any Source

```bash
hikma skill install owner/repo my-skill          # one skill
hikma skill install owner/repo                   # pick from a list
hikma skill install owner/repo my-skill --ref v1 # pin a tag, branch, or commit
```

A source can be GitHub `owner/repo`, any git URL, or a local path. Public sources need only `git`.

## Configure Your Registry

```bash
hikma config set registry hasankhatib/ai
hikma config set registry hasankhatib/ai --project   # share with your team via .hikma/config.json
```

With a registry set, bare names work: `hikma skill install my-skill`. Override it for one command with `--registry`.

## Validating

```bash
hikma registry validate .          # a registry checkout
hikma registry validate owner/repo # any repo
```

Each skill must have `SKILL.md` frontmatter with a `name` matching its folder and a `description`, a folder name that fits your naming setting, and no template placeholders. `push` runs the same checks first.

## Publishing

`hikma skill push <name>` opens a pull request against your configured registry, which must be a GitHub `owner/repo`. It prints the target registry and where the setting came from, then asks you to confirm (`--yes` skips the prompt). Pushing requires the `gh` CLI.

- With write access, a `skill/<name>` branch is pushed to the registry.
- Without it, `hikma` forks the registry and pushes to your fork.
- Nothing is force-pushed. Pushing again updates the open PR with a new commit; if the earlier PR was merged or closed, a new branch is used.

Updating `.github/CODEOWNERS` in the registry is opt-in with `--codeowners`.

## Lockfile

Installs are recorded in `.hikma/lock.json` (source, commit, file hashes). Commit it to make installs reproducible. `hikma skill update` uses the recorded source and stops if you edited an installed skill, unless you pass `--force`.

## Examples

```bash
# Public registry on GitHub
hikma config set registry hasankhatib/ai

# Private registry: works with your existing git or gh credentials
hikma config set registry my-org/private-skills --project

# Local registry, handy while authoring skills
hikma config set registry ./path/to/skills-repo
hikma skill install my-skill
```

Pushing (`hikma skill push`) always needs a GitHub `owner/repo` registry. A local or non-GitHub registry is read-only.
