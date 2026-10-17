#!/usr/bin/env bash
# Installs hikma's shell completion into a throwaway home, checks the shell
# actually registers it, then installs again to confirm nothing changes.
#
#   scripts/test-completion.sh <bash|zsh|fish|pwsh>
#
# hikma must be on PATH. BASH_BIN picks the bash to test (macOS ships 3.2,
# which cannot run cobra's bash completion).
set -euo pipefail

shell=${1:?usage: test-completion.sh bash|zsh|fish|pwsh}
command -v hikma >/dev/null || { echo "hikma is not on PATH" >&2; exit 2; }

home=$(mktemp -d)
export HOME=$home USERPROFILE=$home XDG_CONFIG_HOME=$home/.config
unset SHELL
fail() { echo "FAIL ($shell): $*" >&2; exit 1; }
sum() { if [ -f "$1" ]; then cksum <"$1"; else echo missing; fi; }

# Run the installer twice: the first writes, the second must change nothing.
install_twice() { # <rc-or-file> <installer shell arg>
  local target=$1 arg=$2 before after out
  hikma completion --install --yes "$arg" >/dev/null || fail "install exited non-zero"
  [ -f "$target" ] || fail "expected $target to exist"
  before=$(sum "$target")
  out=$(hikma completion --install --yes "$arg") || fail "second install exited non-zero"
  after=$(sum "$target")
  [ "$before" = "$after" ] || fail "second install changed $target"
  case $out in *already*) ;; *) fail "second install did not say it was already installed: $out" ;; esac
  echo "installed into $target; a second run changed nothing"
}

case $shell in
bash)
  bash_bin=${BASH_BIN:-bash}
  major=$("$bash_bin" -c 'echo "${BASH_VERSINFO[0]}"')
  if [ "$major" -lt 4 ]; then
    echo "SKIP: $("$bash_bin" --version | head -1) is older than 4.1; completion needs bash 4.1+ and bash-completion"
    exit 0
  fi
  rc=$home/.bashrc
  [ "$(uname -s)" = Darwin ] && rc=$home/.bash_profile
  # Real users load bash-completion from their profile; do the same here.
  bc=
  for f in "$(brew --prefix 2>/dev/null)/etc/profile.d/bash_completion.sh" /usr/share/bash-completion/bash_completion /etc/bash_completion; do
    [ -f "$f" ] && { bc=$f; break; }
  done
  install_twice "$rc" bash
  out=$("$bash_bin" --norc --noprofile -c "source '$rc'; complete -p hikma" 2>&1) || fail "bash did not register completion: $out"
  case $out in *__start_hikma*) ;; *) fail "unexpected registration: $out" ;; esac
  # Ask for completions of 'hikma sk' the way bash would.
  if [ -z "$bc" ]; then
    # Git Bash ships no bash-completion, which cobra's bash script needs, so only registration can be checked.
    case $(uname -s) in
      MINGW*|MSYS*|CYGWIN*) echo "registered; bash-completion is not available here (Git Bash), so completion itself is not checked"; echo "OK ($shell)"; exit 0 ;;
    esac
    fail "bash-completion is not installed, so completion cannot be checked"
  fi
  words=$("$bash_bin" --norc --noprofile -c "source '$bc'; source '$rc'; COMP_WORDS=(hikma sk); COMP_CWORD=1; COMP_LINE='hikma sk'; COMP_POINT=\${#COMP_LINE}; __start_hikma; printf '%s\n' \"\${COMPREPLY[@]}\"" 2>&1) || true
  case $words in *skill*) echo "bash completes 'hikma sk' to: $words" ;; *) fail "bash did not complete 'hikma sk': $words" ;; esac
  ;;
zsh)
  printf 'autoload -U compinit && compinit -u\n' >"$home/.zshrc"
  install_twice "$home/.zshrc" zsh
  out=$(ZDOTDIR=$home zsh -ic 'print -r -- ${_comps[hikma]}' 2>&1) || fail "zsh failed: $out"
  case $out in *_hikma*) echo "zsh registered _hikma" ;; *) fail "zsh did not register hikma: $out" ;; esac
  ;;
fish)
  install_twice "$home/.config/fish/completions/hikma.fish" fish
  out=$(fish -c "complete -C 'hikma sk'" 2>&1) || fail "fish failed: $out"
  case $out in *skill*) echo "fish completes 'hikma sk' to: $(echo "$out" | head -1)" ;; *) fail "fish did not complete 'hikma sk': $out" ;; esac
  ;;
pwsh)
  command -v pwsh >/dev/null || { echo "SKIP: pwsh is not installed"; exit 0; }
  profile=$(pwsh -NoProfile -Command '$PROFILE' | tr -d '\r')
  [ -n "$profile" ] || fail "pwsh did not report a profile path"
  # On Windows the profile lives under Documents, not in the throwaway home: start clean.
  rm -f "$profile"
  install_twice "$profile" powershell
  out=$(pwsh -Command "(TabExpansion2 'hikma sk' 8).CompletionMatches.CompletionText -join ','" 2>&1) || fail "pwsh failed: $out"
  case $out in *skill*) echo "pwsh completes 'hikma sk' to: $out" ;; *) fail "pwsh did not complete 'hikma sk': $out" ;; esac
  rm -f "$profile"
  ;;
*) fail "unknown shell $shell" ;;
esac
echo "OK ($shell)"
