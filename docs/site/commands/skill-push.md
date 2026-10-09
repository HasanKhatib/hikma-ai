# hikma skill push

Publish a local skill to your registry as a pull request.

```bash
hikma skill push <name> [flags]
```

Requires the GitHub CLI (`gh`), signed in. The registry must be a GitHub `owner/repo`.

## What happens

1. The skill is validated (the same checks as [`registry validate`](/commands/registry-validate)).
2. **The target registry is shown with where the setting came from, and you confirm it.**

   ```text
   You are pushing to registry: owner/repo (user config)
     skill: deploy-helper (.claude/skills/deploy-helper/)
   ? Push to owner/repo? (y/N)
   ```

3. With write access, a `skill/<name>` branch is pushed to the registry. Without it, Hikma forks the registry and pushes to your fork.
4. A pull request is opened against the registry's default branch.

Nothing is force-pushed:

- Pushing again while the pull request is open adds a commit to it.
- If the earlier pull request was merged or closed, a new branch is used.
- If nothing changed, nothing is pushed.

## Flags

| Flag | Description |
|---|---|
| `-y`, `--yes` | Skip the confirmation prompt (required without a terminal) |
| `--codeowners` | Also add the skill owner to `.github/CODEOWNERS` in the registry |
| `--registry <source>` | Registry to publish to, for this command |
| `--agent <list>` | Which agent's folder to read the skill from |

Executable files keep their executable bit, and binary files are preserved.
