# hikma completion

Generate a shell completion script.

```bash
hikma completion [bash|zsh|fish|powershell]
```

Installed with Homebrew (v0.4.1 and later)? Completions are set up for you. For bash and zsh, Homebrew's completion folders must be on your shell's path: see [Homebrew shell completion](https://docs.brew.sh/Shell-Completion). Otherwise follow the steps for your shell.

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
