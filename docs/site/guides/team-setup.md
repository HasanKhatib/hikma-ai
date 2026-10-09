# Set up a team repo

Share the agent setup through the repository so everyone who clones it gets the same instructions and skills.

## One command

From the repository root:

```bash
hikma init --agent claude,codex --registry owner/repo --skill deploy-helper
git add .
git commit -m "Set up agent instructions and skills"
```

This creates:

```text
AGENTS.md                 project rules, read by every agent
.hikma/config.json        agents and registry
.hikma/lock.json          what was installed, from where
.claude/skills/...        skills for Claude
.agents/skills/...        skills for Codex, Copilot, OpenCode
```

## What teammates get

After cloning, `hikma skill install other-skill` installs into every agent listed in `.hikma/config.json`, from the project's registry, with no flags. Their personal `agent` default does not override a project's `agents` list.

## On a fresh clone

If the skills folders are not committed (for example they are git-ignored), restore them from the lockfile:

```bash
hikma sync
```

## Updating

```bash
hikma skill update --all
```

Review the printed changes, especially anything under `scripts/`, and commit the result. See [Lockfile and updates](/guides/lockfile).

## Edit the instructions

`AGENTS.md` is generated once with placeholders for build commands and invariants. Fill those in; Hikma will not overwrite the file unless you pass `--force`.

## Settings you can share

| Setting | Command |
|---|---|
| Agents | `hikma config set agents claude,codex --project` |
| Registry | `hikma config set registry owner/repo --project` |
| Naming rule | `hikma config set naming kebab-case --project` |
