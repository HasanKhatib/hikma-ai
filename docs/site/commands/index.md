# Commands

## `hikma init`

Set up a repository for one or more agents: writes their instruction files, records the agents and registry in `.hikma/config.json`, and installs the requested skills into every agent's skills folder. Agents sharing a folder (`codex`, `copilot`, `opencode`) are written once. Without a terminal it runs non-interactively from flags.

```bash
hikma init --agent claude,codex --registry hasankhatib/ai --skill agentkan
```

## `hikma config`

Read and write configuration. Values resolve from flag, `HIKMA_*` env var, project config (`.hikma/config.json`), then user config.

```bash
hikma config list                       # every value and where it came from
hikma config get registry
hikma config set registry hasankhatib/ai
hikma config set agents claude,codex --project   # install into every listed agent's folder
hikma config set agent claude                    # personal default
hikma config set naming kebab-case      # or: loose (default); applies to create and push
hikma config unset registry
hikma config path
```

## `hikma skill list`

List skills in the configured registry or in any repo.

```bash
hikma skill list
hikma skill list owner/repo
```

## `hikma skill install`

Install a skill into the selected agent's skill folder.

```bash
hikma skill install my-skill               # from the configured registry
hikma skill install owner/repo my-skill    # from any repo
hikma skill install owner/repo             # pick interactively
hikma skill install owner/repo my-skill --ref v1.2
```

## `hikma skill update`

Update skills from the source recorded in `.hikma/lock.json`.

```bash
hikma skill update my-skill
hikma skill update --all
```

## `hikma skill create`

Create a local skill scaffold.

```bash
hikma skill create my-skill --description "What this skill helps with"
```

## `hikma skill push`

Open a pull request against your configured registry after confirming the target.

```bash
hikma skill push my-skill
hikma skill push my-skill --yes
```

## `hikma doctor`

Check that `git` is available (and `gh`, for push) and show the active agent and registry.
