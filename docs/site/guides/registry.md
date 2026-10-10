# Run a registry

A registry is a Git repository of skills that you choose as your default source and as the target of `hikma skill push`. There is nothing to host: Hikma reads it with `git`, and publishing is a pull request.

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
      - uses: HasanKhatib/hikma-ai@v0.2.0
        with:
          version: v0.2.0
```

| Input | Default | Description |
|---|---|---|
| `path` | `.` | Registry directory to validate |
| `version` | `latest` | Hikma release to install. Pin it, so a new release cannot change your CI |
| `naming` | from `.hikma/config.json` | `loose` or `kebab-case` |

The action needs Hikma v0.2.0 or later, the first release with `--format github`. It runs on Linux, macOS, and Windows runners. Without the action, run `hikma registry validate . --format github` in any step after installing Hikma.

## Publishing

`hikma skill push <name>` opens a pull request against the registry. See [skill push](/commands/skill-push) for the confirmation prompt, fork behavior, and update behavior.

## Keep it neutral

Hikma ships with no registry and no default. Installing from a repository is always something you chose, and every `install` prints the source it used.
