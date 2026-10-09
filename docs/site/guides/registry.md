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

Run this in your registry's CI so every pull request is checked. It exits non-zero when any skill is invalid.

## Publishing

`hikma skill push <name>` opens a pull request against the registry. See [skill push](/commands/skill-push) for the confirmation prompt, fork behavior, and update behavior.

## Keep it neutral

Hikma ships with no registry and no default. Installing from a repository is always something you chose, and every `install` prints the source it used.
