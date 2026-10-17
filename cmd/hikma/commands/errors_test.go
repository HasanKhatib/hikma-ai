package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestFriendlyErrors(t *testing.T) {
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"completion", "zshs"}, []string{`unknown shell "zshs"`, "Did you mean: hikma completion zsh?"}},
		{[]string{"completion"}, []string{"which shell?", "bash|zsh|fish|powershell"}},
		{[]string{"completion", "zsh", "bash"}, []string{"takes one shell name"}},
		{[]string{"skill", "remove"}, []string{"missing argument for 'hikma skill remove'", "Usage: hikma skill remove <name>"}},
		{[]string{"skill", "install", "a", "b", "c"}, []string{"wrong number of arguments (got 3)", "Usage: hikma skill install"}},
		{[]string{"sync", "x"}, []string{`unexpected argument "x"`, "Usage: hikma sync"}},
		{[]string{"sync", "--globl"}, []string{"unknown flag: --globl", "did you mean --global?", "Usage: hikma sync"}},
		{[]string{"skill", "instal"}, []string{`unknown command "instal"`, "Did you mean 'hikma skill install'?"}},
		{[]string{"bogus"}, []string{`unknown command "bogus" for 'hikma'`, "hikma --help"}},
	}
	for _, tc := range cases {
		cmd := Root()
		cmd.SetArgs(tc.args)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		err := cmd.Execute()
		if err == nil {
			t.Errorf("%v: expected an error", tc.args)
			continue
		}
		for _, w := range tc.want {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("%v: error %q does not contain %q", tc.args, err.Error(), w)
			}
		}
	}
}

func TestCompletionWritesScript(t *testing.T) {
	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		var out bytes.Buffer
		cmd := Root()
		cmd.SetArgs([]string{"completion", sh})
		cmd.SetOut(&out)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%s: %v", sh, err)
		}
		if !strings.Contains(out.String(), "hikma") {
			t.Errorf("%s: script not written to the command's output", sh)
		}
	}
}
