# The menu

Run `hikma` with no arguments in a terminal and it opens a menu. It is the guided way to do everything on this site: it asks for what is missing, runs the same command you could type, and prints that command afterwards.

```text
$ hikma
╭──────────╮
│ >_       │  hikma
│          │  skills follow you
╰──────────╯

registry: acme/skills | agent: claude | 3 installed here
Ctrl-C or Esc inside a task comes back here. At this menu they quit.

? What do you want to do?
> Install skills from your registry
  Browse skills in your registry
  Update installed skills
  Check installed skills for local edits
  Create a new skill
  Publish skills to your registry
  Setup and settings >
  Quit
```

If a skills folder cannot be read, the last part says `installed here: unknown (cannot read the skills folders)` instead of `0`, so an unreadable folder is not mistaken for an empty one.

When no registry is set (in the environment, the project, or your user config), the menu first asks for one: type `owner/repo` and it runs `hikma config registry`, or press Enter or Esc to skip to the menu. When a registry is set, it does not ask.

Pick an entry and Hikma asks for anything it needs, then runs the same command you could type yourself. It asks only for what is missing: the registry when none is set, the agent when none is chosen, a skill from this project's agent folders when you publish or update, and the agents to set up for a new project. Agent, registry, project setup, completion, and the setup check live under "Setup and settings".

## Installing skills

Install reads the registry (with a spinner), lists the skills you do not have yet, and lets you tick as many as you want. Long lists can be filtered by typing `/`, and `ctrl+a` ticks them all. Skills you already have for your agent are listed last, crossed out and marked `[installed]`, and ticking one does nothing: reinstalling would need `--force` and replace local edits, so use Update installed skills for those. A copy installed for a different agent does not count. When every skill is already installed, the menu says so, lists them, and tells you how to refresh them; it is not an error. Before anything is written it shows a plan and asks once:

```text
Will install 2 skills from acme/skills
  foo  Lint a repo
  bar  Format docs
Into .claude/skills (agent: claude)
Install these? Yes / No
```

It then runs one `hikma skill install --skill=foo --skill=bar` and prints the result and that command, so you can run it yourself next time:

```text
Installed 2 of 2
  ok      foo
  ok      bar

Equivalent command:
  hikma skill install --skill=foo --skill=bar
```

If some skills fail, the others are still installed, the result names each one that failed, and "What next?" offers to retry just those. Skills that failed only because they are already installed are not offered again, since trying again would fail the same way; the menu says how many there were and points you to Update installed skills. After a retry works, the equivalent command printed is the whole install, not just the retried skills.

## Creating and publishing

**Create a new skill** asks for the name and a one-line description (Enter to fill it in later), and links the [guide on writing a skill](/guides/write-a-skill). It prints the command it ran, with `--description` in it.

**Publish skills to your registry** reads your registry first and marks the skills that already exist there with `[exists]`, so you can see what publishing again would update. If the registry cannot be read, for example because it is empty, you can still publish; nothing is marked.

Publishing takes several skills at once:

- Tick as many as you like. Skills from every agent folder of the project are listed, with the folder shown beside each name when there is more than one, for example `video-pixel-scene  (.claude/skills)`.
- A skill outside your agent's folder is published with `--agent`, so the command finds it.
- You confirm once, with the registry and how it will be published (a direct commit or a pull request).
- Each skill is then published by its own `hikma skill push <name> --yes`, so each one is its own commit or pull request.
- If one fails, the menu lists the ones that did not run.
- The same skill ticked from two folders is refused: tick one copy.

**Check installed skills for local edits** runs `hikma skill verify`. Before it runs, the menu says what it does: it compares each installed skill with the hashes recorded in `.hikma/lock.json` at install time, offline, and changes nothing. `FAIL` means the skill was edited, or files were added or removed, since. Skills that Hikma did not install have no record and are not checked.

Lists where you tick several options show a checkbox, `[ ]` or `[x]`, so a ticked option reads the same without color.

## What the menu does and does not do

- It is a front end. Every entry runs an ordinary `hikma` command, so the confirmation questions before an update or a file write are the ones that command always asks. Publishing asks once in the menu instead, for all the ticked skills, and the command it prints carries `--yes`.
- It clears the screen and stays open. After each task it shows the result and asks "What next?": go back to the menu or quit. Errors are shown there too, so a failed task never closes the menu.
- The equivalent command is printed after a task that worked, not after a failure or a cancel. When some skills of an install fail, it prints the command that ran and the command that retries only the ones that can still work.
- Ctrl-C or Esc inside a task goes back to the menu without doing anything: at a question, while a registry is being read, or while a command runs (a second Ctrl-C while it is stopping ends hikma). The same goes for "What next?". At the menu itself they leave. On a long list Esc clears the filter instead, and Ctrl-C goes back.
- The cursor stays on the entry you used last.
- A mistyped registry asks for another one without asking for your agent again.
- Clearing keeps your scrollback. In accessible mode (`ACCESSIBLE` set) and under `TERM=dumb` the screen is not cleared, the banner is drawn once, and lists are answered by number.

## Scripts and CI

The menu opens only when both input and output are a terminal. When either is piped or redirected, `hikma` with no arguments prints help and exits, and nothing waits for input.

Pass `--no-input` to any command to rule out prompts even in a terminal. A command that would have asked fails instead and names the flag to pass, such as `--yes` to confirm. Bare `hikma --no-input` prints help instead of opening the menu.

## Color

The banner and the command output use color in a terminal. Set `NO_COLOR=1` or `TERM=dumb`, or pass `--no-color`, to turn it off. Piped output is always plain, and no message relies on color alone.

## Next

- [Command reference](/commands/)
- [Troubleshooting](/guides/troubleshooting)
