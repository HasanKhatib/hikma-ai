package e2e

import (
	"os/exec"
	"testing"
)

// runPlain runs hikma with piped output, the way a script sees it.
func runPlain(t *testing.T, sb sandbox, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(hikmaBinary(t), args...)
	cmd.Dir = sb.project
	cmd.Env = sb.env
	out, err := cmd.CombinedOutput()
	return string(out), err
}
