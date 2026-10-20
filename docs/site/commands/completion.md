# hikma completion

Install shell completion, or print the script.

```bash
hikma completion                 # detect your shell, show what changes, ask, install
hikma completion --install zsh   # same, for a shell you name
hikma completion zsh             # only print the script
```

The installer shows the file it will change and asks before writing (`--yes` skips the question). zsh and bash get a marked `source` line appended to `~/.zshrc` or `~/.bashrc` (`~/.bash_profile` on macOS), skipped quietly if `hikma` is ever removed; fish gets `~/.config/fish/completions/hikma.fish`; PowerShell gets a line in your `$PROFILE`. Running it twice changes nothing: if the line is already there it says so, and tells you what it cannot see for you. A line in `~/.zshrc` only works in terminal windows opened after it was added, so in an open window run `source ~/.zshrc` or open a new one. For zsh it also checks that `compinit` (or oh-my-zsh) comes before the hikma line; without it zsh prints `command not found: compdef` and cannot register the completion, and the message says to move the lines below it.

Bash completion needs bash 4.1 or newer with the `bash-completion` package loaded before the line the installer adds; macOS ships bash 3.2, and Git Bash on Windows has no `bash-completion`, so use PowerShell there. To do it by hand, follow the steps below.

Installed with Homebrew? Completions are set up for you. For bash and zsh, Homebrew's completion folders must be on your shell's path: see [Homebrew shell completion](https://docs.brew.sh/Shell-Completion). Otherwise follow the steps for your shell.

::: code-group

```bash [bash]
source <(hikma completion bash)
# persist:
hikma completion bash > /etc/bash_completion.d/hikma
```

```bash [zsh]
# Add to ~/.zshrc, after your compinit line (or oh-my-zsh is loaded):
source <(hikma completion zsh)
```

Or write it to a completion folder you own and make sure that folder is on `fpath` before `compinit` runs:

```bash [zsh (file)]
mkdir -p ~/.zsh/completions
hikma completion zsh > ~/.zsh/completions/_hikma
# in ~/.zshrc, before compinit:
fpath=(~/.zsh/completions $fpath)
autoload -U compinit && compinit
```

```bash [fish]
hikma completion fish > ~/.config/fish/completions/hikma.fish
```

```powershell [PowerShell]
hikma completion powershell | Out-String | Invoke-Expression
```

:::

If completion does not appear, open a new shell. If it still does not, delete the completion cache (`rm -f ~/.zcompdump*`) and open another.

## Mistyped commands and flags

Hikma names what is wrong and what to type instead:

```text
$ hikma completion zshs
unknown shell "zshs". Supported: bash, zsh, fish, powershell. Did you mean: hikma completion zsh?
```
