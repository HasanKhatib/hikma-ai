# Commands

## `hikma init`

Scaffold AI-agent configuration files into a repository.

```bash
hikma init --agent claude --registry hasankhatib/ai
```

## `hikma config agent`

Get or set the active agent layout.

```bash
hikma config agent claude
```

## `hikma config registry`

Get or set the active external registry.

```bash
hikma config registry hasankhatib/ai
```

## `hikma skill list`

List skills from the configured registry.

```bash
hikma skill list
hikma skill list --registry hasankhatib/ai
```

## `hikma skill install`

Install a skill from the configured registry into the selected agent path.

```bash
hikma skill install agentkan --agent claude
```

## `hikma skill create`

Create a local skill scaffold.

```bash
hikma skill create my-skill --description "What this skill helps with"
```

## `hikma skill push`

Open a pull request against the configured registry repository.

```bash
hikma skill push my-skill --registry hasankhatib/ai
```
