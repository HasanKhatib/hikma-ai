# Run a registry

A registry is a Git repository of skills that you choose as your default source and as the target of `hikma skill push`. There is nothing to host: Hikma reads it with `git`, and publishing is a commit if you own it or a pull request if you do not.

## Create one

Any repository with skills in the [standard layout](/guides/skill-format#where-skills-live) works:

```text
skills/
  deploy-helper/
    SKILL.md
  release-notes/
    SKILL.md
```

Point Hikma at it:

```bash
hikma config set registry owner/repo            # for you
hikma config set registry owner/repo --project  # for everyone on a repo
```

Bare names now install from it: `hikma skill install deploy-helper`.

## Source values

```text
owner/repo
https://github.com/owner/repo.git
git@github.com:owner/repo.git
./path/to/local/registry
```

## Public, private, and local

| Registry | Works with |
|---|---|
| Public GitHub repo | `git` only |
| Private GitHub repo | your git credentials, or the GitHub CLI as a fallback |
| Other git host | read-only; `skill push` needs GitHub |
| Local path | read-only; handy while writing skills |

## Check it

```bash
hikma registry validate .
```

It exits non-zero when any skill is invalid, so it works as a pull request check.

## Check it on every pull request

Hikma ships a GitHub Action that installs a pinned release and validates the registry. Problems show up as annotations on the files in the pull request.

```yaml
# .github/workflows/validate.yml
name: Validate skills
on: pull_request

permissions:
  contents: read

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: HasanKhatib/hikma-ai@v1.1.0
        with:
          version: v1.1.0
```

| Input | Default | Description |
|---|---|---|
| `path` | `.` | Registry directory to validate |
| `version` | `latest` | Hikma release to install. Pin it, so a new release cannot change your CI |
| `naming` | from `.hikma/config.json` | `loose` or `kebab-case` |

It runs on Linux, macOS, and Windows runners. Without the action, run `hikma registry validate . --format github` in any step after installing Hikma.

## Publishing

`hikma skill push <name>` and [`hikma skill import`](/commands/skill-import) publish the same way. See [skill push](/commands/skill-push) for the confirmation prompt, fork behavior, and update behavior.

## How publishing decides

Hikma reads this from GitHub each time and never stores it, because write access differs from person to person and branch rules change. Nothing to configure, and nothing asked at `init`.

| What GitHub says | What push and import do |
|---|---|
| You can write, and no rule requires pull requests | Commit to the default branch, whatever its name (`main`, `master`, ...) |
| A rule on the default branch requires pull requests | Open a pull request, and say why |
| You cannot write | Fork the registry and open a pull request from the fork |
| You pass `--pr` | Open a pull request even if you could commit |

Every confirmation says which of these will happen, in words, before anything is written. `hikma init --registry ...` and `hikma doctor` print the same for your registry:

```text
Registry owner/registry: push and import commit straight to main (you can write, and no rule requires a pull request); add --pr to open one instead
```

**To review everything that enters the registry**, require pull requests with a branch protection rule or a ruleset on the default branch. Hikma then opens a pull request by itself. Rulesets are visible to Hikma. Classic branch protection is not, so with it a direct commit is rejected and the error suggests `--pr`.

## Importing skills

`hikma skill import <source> [<name>]` brings a skill you found elsewhere into the registry, with a review first and its origin recorded in `skills/<name>/.hikma-origin.json`. It never overwrites an existing skill without `--force`. In a registry with CI, [`registry validate`](/commands/registry-validate) accepts the origin file like any other file in a skill folder. See [skill import](/commands/skill-import).

## Keep it neutral

Hikma ships with no registry and no default. Installing from a repository is always something you chose, and every `install` prints the source it used.
