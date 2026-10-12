# hikma skill push

Publish a local skill to your registry.

```bash
hikma skill push <name> [flags]
```

Requires the GitHub CLI (`gh`), signed in. The registry must be a GitHub `owner/repo`.

## What happens

1. The skill is validated with the same checks as [`registry validate`](/commands/registry-validate). Errors stop the push. Unfilled template placeholders (such as "Replace with clear steps for the agent.") are only warnings here, so you can push a draft for review; `registry validate` still fails them, so a registry's CI will ask you to fill them in before merge.
2. **The target registry is shown with where the setting came from, and you confirm it.**

   ```text
   You are pushing to registry: owner/repo (user config)
     skill: deploy-helper (.claude/skills/deploy-helper/)
   ? Push to owner/repo? (y/N)
   ```

3. **With write access, the skill is committed straight to the registry's default branch.** No branch, no pull request: the registry is yours, so publishing is one step. The confirmation shows this before anything is written. If the branch is protected and rejects the commit, Hikma says so and suggests `--pr`.
4. **Without write access**, Hikma forks the registry, pushes a `skill/<name>` branch to your fork, and opens a pull request against the registry.
5. **With `--pr`**, you get the pull request flow even with write access, for a registry where changes are reviewed. A `skill/<name>` branch is pushed to the registry and a pull request is opened.

Nothing is force-pushed:

- Pushing again while a `--pr` pull request is open adds a commit to it.
- If the earlier pull request was merged or closed, a new branch is used.
- If nothing changed, nothing is pushed.

## Flags

| Flag | Description |
|---|---|
| `-y`, `--yes` | Skip the confirmation prompt (required without a terminal) |
| `--pr` | Open a pull request even if you can write to the registry |
| `--codeowners` | Also add the skill owner to `.github/CODEOWNERS` in the registry |
| `--registry <source>` | Registry to publish to, for this command |
| `--agent <list>` | Which agent's folder to read the skill from |

Executable files keep their executable bit, and binary files are preserved.
