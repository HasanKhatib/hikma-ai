# hikma completion

Generate a shell completion script.

```bash
hikma completion [bash|zsh|fish|powershell]
```

::: code-group

```bash [bash]
source <(hikma completion bash)
# persist:
hikma completion bash > /etc/bash_completion.d/hikma
```

```bash [zsh]
hikma completion zsh > "${fpath[1]}/_hikma"
autoload -U compinit && compinit
```

```bash [fish]
hikma completion fish > ~/.config/fish/completions/hikma.fish
```

```powershell [PowerShell]
hikma completion powershell | Out-String | Invoke-Expression
```

:::
