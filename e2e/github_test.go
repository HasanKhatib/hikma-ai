package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGitHubRoundTrip runs the real binary against a real, throwaway GitHub
// registry: create a skill, push it, install it into a second project, verify,
// change it upstream, update, and verify again. It needs the network and a
// token that can write to the registry, so it only runs when HIKMA_E2E_REPO
// (owner/repo of the throwaway registry) and GH_TOKEN are set. The skill gets a
// unique name, and the test deletes it from the registry afterwards.
func TestGitHubRoundTrip(t *testing.T) {
	repo := os.Getenv("HIKMA_E2E_REPO")
	token := os.Getenv("GH_TOKEN")
	if repo == "" || token == "" {
		t.Skip("set HIKMA_E2E_REPO=owner/repo and GH_TOKEN to run the GitHub round trip")
	}
	if _, err := exec.LookPath("gh"); err != nil {
		t.Fatal("gh is not on PATH")
	}
	name := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	t.Cleanup(func() { deleteSkill(t, repo, token, name) })

	// A registry is always user-configured, so pass it per command and keep the
	// test off the host's git credential helpers (a locked keychain would hang).
	envFor := func(sb sandbox) []string {
		return sb.with("GH_TOKEN="+token, "GIT_TERMINAL_PROMPT=0",
			"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=credential.helper", "GIT_CONFIG_VALUE_0=")
	}
	hikma := func(sb sandbox, args ...string) string {
		t.Helper()
		cmd := exec.Command(hikmaBinary(t), append(args, "--no-input")...)
		cmd.Dir = sb.project
		cmd.Env = envFor(sb)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("hikma %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return string(out)
	}

	author, other := newSandbox(t), newSandbox(t)

	hikma(author, "skill", "create", name, "--description", "end-to-end test skill", "--team", "e2e", "--agent", "claude")
	if _, err := os.Stat(filepath.Join(author.project, ".claude", "skills", name, "SKILL.md")); err != nil {
		t.Fatalf("skill create wrote nothing: %v", err)
	}
	hikma(author, "skill", "push", name, "--agent", "claude", "--registry", repo, "--yes")

	hikma(other, "skill", "install", name, "--agent", "claude", "--registry", repo, "--yes")
	if out := hikma(other, "skill", "verify", "--json"); !strings.Contains(out, `"ok": true`) {
		t.Fatalf("verify after install:\n%s", out)
	}

	skill := filepath.Join(author.project, ".claude", "skills", name, "SKILL.md")
	body, err := os.ReadFile(skill)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skill, append(body, []byte("\nSecond version.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	hikma(author, "skill", "push", name, "--agent", "claude", "--registry", repo, "--yes")

	hikma(other, "skill", "update", name, "--agent", "claude", "--yes")
	got, err := os.ReadFile(filepath.Join(other.project, ".claude", "skills", name, "SKILL.md"))
	if err != nil || !strings.Contains(string(got), "Second version.") {
		t.Fatalf("update did not bring the new version (err %v):\n%s", err, got)
	}
	if out := hikma(other, "skill", "verify", "--json"); !strings.Contains(out, `"ok": true`) {
		t.Fatalf("verify after update:\n%s", out)
	}
}

// deleteSkill removes the test skill's files from the registry's default
// branch. It is best effort: a leftover skill in a throwaway repo is harmless.
func deleteSkill(t *testing.T, repo, token, name string) {
	t.Helper()
	gh := func(args ...string) ([]byte, error) {
		cmd := exec.Command("gh", args...)
		cmd.Env = append(os.Environ(), "GH_TOKEN="+token)
		return cmd.Output()
	}
	var walk func(dir string)
	walk = func(dir string) {
		out, err := gh("api", fmt.Sprintf("repos/%s/contents/%s", repo, dir))
		if err != nil {
			return
		}
		var items []struct{ Path, Type, SHA string }
		if json.Unmarshal(out, &items) != nil {
			return
		}
		for _, it := range items {
			if it.Type == "dir" {
				walk(it.Path)
				continue
			}
			if _, err := gh("api", "-X", "DELETE", fmt.Sprintf("repos/%s/contents/%s", repo, it.Path),
				"-f", "message=chore: remove e2e test skill "+name, "-f", "sha="+it.SHA); err != nil {
				t.Logf("cleanup: could not delete %s: %v", it.Path, err)
			}
		}
	}
	walk("skills/" + name)
}
