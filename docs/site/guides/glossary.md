# Glossary

The few terms Hikma uses, in plain words.

| Term | Meaning |
|---|---|
| **Agent** | An AI coding assistant, such as Claude, Codex, Copilot, or OpenCode. Each reads skills from its own folder. |
| **Skill** | A folder with a `SKILL.md` file that teaches an agent how to do one job. See [Skill format](/guides/skill-format). |
| **Registry** | A Git repository where your skills live. You choose it; Hikma has no default and hosts nothing. See [Run a registry](/guides/registry). |
| **Source** | Where a skill comes from: `owner/repo` on GitHub, a git URL, or a local path. A registry is one kind of source. |
| **Install folder** | The folder in your project where an agent looks for skills, such as `.claude/skills`. See [Agents and folders](/guides/agents). |
| **Lockfile** | `.hikma/lock.json`. It records each installed skill, where it came from, and a fingerprint of its files. Commit it so everyone gets the same skills. |
| **Sync** | Install everything the lockfile lists. This is how a new laptop or a teammate catches up. See [sync](/commands/sync). |
| **Verify** | Check that installed files still match the lockfile. See [skill verify](/commands/skill-verify). |
| **Push** | Publish a skill from your project to your registry. It asks first. See [skill push](/commands/skill-push). |
| **Ref** | A branch, tag, or commit to install from, for example `--ref v1.0`. |
| **Project and global** | A project install lives in the repository you are in. A global install (`--global`) lives in your home folder, so every project can use it. |
| **Frontmatter** | The block at the top of `SKILL.md`, between two `---` lines, that holds the skill's name and description. |
| **Attestation** | A signed statement from GitHub that a release file was built by Hikma's release workflow. Optional to check. |
