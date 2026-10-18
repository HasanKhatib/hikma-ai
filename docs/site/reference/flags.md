# Flags and terminal behavior

Flags that work on every command, flags many commands share, and how hikma decides about color and prompts.

## Global flags

| Flag | Meaning |
|---|---|
| `--no-input` | Never prompt. A command that would ask fails and names the flag to pass instead |
| `--no-color` | Turn color off, the same as setting `NO_COLOR` |
| `-h`, `--help` | Show help for the command |
| `--version` | (on `hikma`) Print the version |

## Shared flags

These appear on several commands with the same meaning.

| Flag | Commands | Meaning |
|---|---|---|
| `-g`, `--global` | `skill install`, `update`, `remove`, `adopt`, `verify`, `diff`, `list --installed`, `sync` | Use your home directory and `~/.hikma/lock.json` instead of this project |
| `-y`, `--yes` | `skill install`, `update`, `remove`, `adopt`, `skill push`, `sync`, `config add-agent`, `completion --install` | Skip the confirmation. Required when there is no terminal |
| `--agent <list>` | most skill commands | Limit to these agents, comma-separated |
| `--json` | `skill verify`, `skill diff`, `skill list`, `skill info`, `sync`, `config list` | Print one versioned JSON document on stdout. See [Exit codes and JSON](/reference/output) |
| `--dry-run` | `sync`, `init` | Show what would happen and change nothing |
| `--force` | `skill install`, `skill update`, `skill remove`, `sync`, `init` | Overwrite or remove local edits. Never the default |

Anything that writes outside the project, such as `--global`, shows the target and asks before writing. `--yes` is the only way to skip the question.

## Color

Color and the banner appear only when output goes to a terminal. They are off when any of these is true, and the first match wins:

1. `--no-color` is passed
2. `NO_COLOR` is set to anything (see [no-color.org](https://no-color.org))
3. `TERM=dumb`
4. Output is piped, redirected, or captured

`COLORTERM=truecolor` (or `24bit`) selects the smooth gradient in the banner; without it a fixed palette is used. Nothing relies on color alone: every status also has a word (`ok`, `FAIL`, `modified`).

## Prompts

hikma prompts only when standard input is a terminal and `--no-input` is not passed. Without a terminal it never waits: it prints help or the plan, or fails and names the flag to pass (usually `--yes`). The menu and wizards are front ends over the same commands and print the equivalent `hikma` command when they finish.

`ACCESSIBLE` (any value) and `TERM=dumb` switch the prompts to plain line questions, which suit screen readers and basic terminals.

## Precedence at a glance

| Question | Highest to lowest |
|---|---|
| Is there color? | `--no-color`, then `NO_COLOR`, then `TERM=dumb`, then whether output is a terminal |
| May hikma prompt? | `--no-input`, then whether standard input is a terminal |
| Which agent or registry? | Flag, then `HIKMA_*`, then project config, then user config, then default. See [Configuration](/reference/configuration) |
