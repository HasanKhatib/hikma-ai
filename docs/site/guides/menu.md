# Menu and wizard

Run `hikma` with no arguments in a terminal and it opens a menu:

```text
$ hikma
╭──────────╮
│ >_       │  hikma
│          │  skills follow you
╰──────────╯

? What do you want to do?
> Install a skill from your registry
  Browse skills in your registry
  Update installed skills
  Verify installed skills
  Create a new skill
  Publish a skill to your registry
  Set up this project for your agent
  Choose your agent
  Set your registry
  Install shell completion
  Check your setup
  Quit
```

Pick an entry and Hikma asks for anything it needs, then runs the same command you could type yourself. When it finishes it prints that command:

```text
Equivalent command:
  hikma skill create my-skill
```

Copy it into a script or run it directly next time.

## What the menu does and does not do

- It is a front end. Every entry runs an ordinary `hikma` command, so the confirmation questions before a push, an update, or a file write are the ones that command always asks.
- It runs one task and exits. Run `hikma` again for the next.
- Press Ctrl-C at any question to leave without doing anything.

## Scripts and CI

The menu opens only when both input and output are a terminal. When either is piped or redirected, `hikma` with no arguments prints help and exits, and nothing waits for input.

## Color

The banner and the command output use color in a terminal. Set `NO_COLOR=1` or `TERM=dumb` to turn it off. Piped output is always plain, and no message relies on color alone.
