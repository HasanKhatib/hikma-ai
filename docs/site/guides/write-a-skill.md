# Write and edit a skill

A skill is a folder of instructions your AI agent reads. This page takes you from nothing to a skill you can share. For the exact file layout and fields, see [Skill format](/guides/skill-format).

## 1. Create it

```bash
hikma skill create deploy-helper --description "Deploy a service and verify its health. Use when asked to roll out or roll back."
```

Run `hikma` and pick **Create a new skill** to be asked for the name and the description instead. Hikma creates `deploy-helper/` in your agent's skills folder (for Claude, `.claude/skills/deploy-helper/`) with a `SKILL.md` template and empty `scripts/`, `references/`, and `assets/` folders.

## 2. Write the description first

The description is the one line your agent reads to decide whether to use the skill. Agents load only the name and the description up front, and read the rest of `SKILL.md` when the skill matches. So:

- say **what** the skill does and **when** to use it, for example "Use when asked to roll out or roll back";
- include the words a person would actually say, such as "release notes", "pull request", "deploy";
- keep it to a sentence or two.

You can change it any time by editing the `description:` line at the top of `SKILL.md`.

## 3. Edit `SKILL.md`

Open the file and replace the template text:

- **When to use**: the situations that should trigger the skill.
- **Instructions**: clear steps for the agent, in the order it should do them.
- **Boundaries**: what the skill must not do, such as touching production or printing secrets.

Keep the file short (under 500 lines). Put long reference material in `references/` and link to it, and put helper scripts in `scripts/`. A skill can contain scripts your agent may run, so write only what you would be comfortable reading in a review.

## 4. Share it

```bash
hikma skill push deploy-helper
```

`push` checks the skill and refuses to publish template text you have not replaced. It shows the registry it will write to and asks you to confirm. See [skill push](/commands/skill-push).

## Make it better: scripts, references, assets, and hooks

What the folders are for, as the open [Agent Skills specification](https://agentskills.io/specification) describes them:

- **`references/`**: extra documentation the agent reads only when it needs it. Keep each file focused, and link to it directly from `SKILL.md`. The spec and Anthropic both say to keep links one level deep, so the agent reads whole files instead of previewing a chain of them. A long reference file is easier to use with a table of contents at the top (Anthropic suggests this for files over 100 lines).
- **`scripts/`**: code the agent can run. Both Anthropic and OpenAI recommend a script when a step must be deterministic or needs external tooling, and plain instructions otherwise. Say in `SKILL.md` whether the agent should **run** a script ("Run `scripts/validate.py`") or **read** it as reference. A script should be self-contained, handle errors itself with a helpful message instead of leaving the agent to guess, and explain any constant it uses.
- **`assets/`**: files used in the output rather than read as instructions: templates, images, lookup tables, schemas.
- **Hooks**: these are not part of the open spec. In Claude Code, a hook is a shell command that runs at a fixed point in a session, such as before a tool call (`PreToolUse`, which can block it), after one (`PostToolUse`), or when Claude finishes (`Stop`). Unlike a line in `SKILL.md`, a hook always runs, so it suits rules that must not depend on the model remembering them. A Claude Code skill can declare hooks in its frontmatter; they are registered when the skill is invoked. Hooks are specific to Claude Code and other agents may not support them, and `hikma registry validate` warns about fields outside the spec, so keep hooks to skills meant for Claude Code.

Habits both vendors recommend:

- **One job per skill**, with a description that gives the scope and boundaries and puts the main use case and trigger words first (OpenAI).
- **Write the description in the third person**: "Processes Excel files and generates reports", not "I can help you process Excel files" (Anthropic). The description is added to the agent's instructions, and mixed points of view make skills harder to find.
- **Assume the agent is already capable.** Add only what it does not know, such as your conventions, names, and rules. Every line competes with the rest of the conversation for space (Anthropic).
- **Match strictness to risk.** Give exact commands for fragile steps such as a database migration, and a goal for open-ended ones such as a code review (Anthropic).
- **Add a check for risky work**: have the agent make a plan, validate it with a script, then act (Anthropic's plan-validate-execute pattern).
- **Test it.** Run the agent on real tasks without the skill first to find what is missing, write just enough to fix that, and then test real prompts against the description to confirm the skill triggers when it should (Anthropic and OpenAI both describe this). Anthropic's checklist suggests at least three test scenarios, run on each model you use.
- **Read before you install.** Skills can contain scripts, and Anthropic advises installing skills only from sources you trust and reading bundled files and scripts first. Hikma's [`skill import`](/commands/skill-import) shows what a skill contains before it goes into your registry.

To work in every agent, follow the spec's name rules (lowercase letters, digits, and single hyphens, matching the folder name) with `hikma config set naming kebab-case`, and keep descriptions under 1,024 characters.

## Further reading

Checked on 2026-10-20. These pages move, so search for the title if a link fails.

- Anthropic: [Skill authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices): the most complete guide, with a checklist.
- Anthropic: [Equipping agents for the real world with Agent Skills](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills) (October 2025): why skills are built the way they are.
- [Agent Skills specification](https://agentskills.io/specification): the open format, with field limits and a validator.
- OpenAI: [Build skills](https://learn.chatgpt.com/docs/build-skills): skills in Codex and ChatGPT, including `agents/openai.yaml` and how to stop a skill from triggering on its own.
- Claude Code: [Extend Claude with skills](https://code.claude.com/docs/en/skills): the extra frontmatter fields Claude Code reads, such as `disable-model-invocation` and `allowed-tools`.
- Claude Code: [Automate actions with hooks](https://code.claude.com/docs/en/hooks-guide): events, exit codes, and examples.

## Edit a skill you already have

Edit the files in your agent's skills folder directly. For a skill you installed from your registry, `hikma skill diff <name>` shows your local edits and any upstream changes. Publish the new version with `hikma skill push <name>`. To pick up someone else's changes, run `hikma skill update <name>`; Hikma will not overwrite your local edits unless you say so.
