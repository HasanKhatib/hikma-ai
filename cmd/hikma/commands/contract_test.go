package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runSplit runs hikma with stdout and stderr captured apart.
func runSplit(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := Root()
	var o, e bytes.Buffer
	cmd.SetOut(&o)
	cmd.SetErr(&e)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return o.String(), e.String(), err
}

func decode(t *testing.T, stdout string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, stdout)
	}
	if doc["version"] != float64(1) {
		t.Errorf("version = %v, want 1", doc["version"])
	}
	return doc
}

func TestExitCodes(t *testing.T) {
	project(t)
	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{"success", []string{"skill", "verify"}, ExitOK},
		{"unknown flag", []string{"skill", "verify", "--nope"}, ExitUsage},
		{"unknown command", []string{"skil"}, ExitUsage},
		{"missing argument", []string{"skill", "remove"}, ExitUsage},
		{"failure", []string{"skill", "verify", "nope"}, ExitFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runSplit(t, tc.args...)
			if got := ExitCode(err); got != tc.want {
				t.Errorf("ExitCode = %d, want %d (err: %v)", got, tc.want, err)
			}
		})
	}
}

func TestVerifyMismatchExitsThreeAndJSONSaysWhy(t *testing.T) {
	reg := registryRepo(t)
	proj := project(t)
	if out, err := run(t, "skill", "install", reg, "alpha", "--agent", "claude"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}

	stdout, _, err := runSplit(t, "skill", "verify", "--json")
	if err != nil {
		t.Fatal(err)
	}
	doc := decode(t, stdout)
	if doc["ok"] != true || doc["checked"] != float64(1) || doc["differ"] != float64(0) {
		t.Errorf("clean doc: %v", doc)
	}

	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "extra.md"), "x\n")
	stdout, _, err = runSplit(t, "skill", "verify", "--json")
	if ExitCode(err) != ExitMismatch {
		t.Fatalf("ExitCode = %d, want 3 (err: %v)", ExitCode(err), err)
	}
	doc = decode(t, stdout)
	row := doc["skills"].([]any)[0].(map[string]any)
	if doc["ok"] != false || row["status"] != "modified" || row["added"].([]any)[0] != "extra.md" {
		t.Errorf("drift doc: %v", doc)
	}
	if strings.Contains(stdout, "FAIL") || strings.Contains(stdout, "checked,") {
		t.Errorf("human text leaked into JSON stdout:\n%s", stdout)
	}

	if err := os.RemoveAll(filepath.Join(proj, ".claude")); err != nil {
		t.Fatal(err)
	}
	stdout, _, err = runSplit(t, "skill", "verify", "--json")
	if ExitCode(err) != ExitMismatch {
		t.Fatalf("missing folder: ExitCode = %d", ExitCode(err))
	}
	if row := decode(t, stdout)["skills"].([]any)[0].(map[string]any); row["status"] != "missing" {
		t.Errorf("missing doc: %v", row)
	}
}

func TestFrozenSyncDriftExitsThree(t *testing.T) {
	reg := registryRepo(t)
	proj := installFromURL(t, reg, "claude")
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md"), "edited\n")
	_, _, err := runSplit(t, "sync", "--frozen", "--yes")
	if ExitCode(err) != ExitMismatch {
		t.Fatalf("ExitCode = %d, want 3 (err: %v)", ExitCode(err), err)
	}
}

func TestSyncJSONNeedsYesOrDryRun(t *testing.T) {
	project(t)
	_, _, err := runSplit(t, "sync", "--json")
	if ExitCode(err) != ExitUsage {
		t.Fatalf("ExitCode = %d, want 2 (err: %v)", ExitCode(err), err)
	}
}

func TestSyncJSONPlanAndResult(t *testing.T) {
	reg := registryRepo(t)
	proj := installFromURL(t, reg, "claude")
	if err := os.RemoveAll(filepath.Join(proj, ".claude")); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runSplit(t, "sync", "--dry-run", "--json")
	if err != nil {
		t.Fatal(err)
	}
	doc := decode(t, stdout)
	row := doc["skills"].([]any)[0].(map[string]any)
	if doc["dryRun"] != true || row["state"] != "restore" || row["result"] != "planned" {
		t.Errorf("dry-run doc: %v", doc)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude")); err == nil {
		t.Error("dry run restored files")
	}

	stdout, stderr, err := runSplit(t, "sync", "--yes", "--json")
	if err != nil {
		t.Fatal(err)
	}
	doc = decode(t, stdout)
	if row := doc["skills"].([]any)[0].(map[string]any); doc["ok"] != true || row["result"] != "restored" {
		t.Errorf("restore doc: %v", doc)
	}
	if !strings.Contains(stderr, "restored") || strings.Contains(stdout, "Done:") {
		t.Errorf("progress should go to stderr only:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md"), "edited\n")
	stdout, _, err = runSplit(t, "sync", "--yes", "--json")
	if err == nil {
		t.Fatal("a skipped skill should make sync incomplete")
	}
	if doc = decode(t, stdout); doc["ok"] != false || doc["skills"].([]any)[0].(map[string]any)["result"] != "skipped" {
		t.Errorf("skipped doc: %v", doc)
	}
}

func TestDiffJSON(t *testing.T) {
	reg := registryRepo(t)
	proj := installFromURL(t, reg, "claude")
	write(t, filepath.Join(proj, ".claude", "skills", "alpha", "extra.md"), "x\n")

	stdout, _, err := runSplit(t, "skill", "diff", "alpha", "--agent", "claude", "--json")
	if err != nil {
		t.Fatal(err)
	}
	row := decode(t, stdout)["skills"].([]any)[0].(map[string]any)
	edits := row["localEdits"].(map[string]any)
	up := row["upstream"].(map[string]any)
	if edits["added"].([]any)[0] != "extra.md" || up["upToDate"] != true || row["missing"] != false {
		t.Errorf("diff doc: %v", row)
	}
	if strings.Contains(stdout, "Local edits:") {
		t.Errorf("human text leaked into JSON stdout:\n%s", stdout)
	}
	_, _, err = runSplit(t, "skill", "diff", "alpha", "--json", "--patch")
	if ExitCode(err) != ExitUsage {
		t.Errorf("--json with --patch: ExitCode = %d", ExitCode(err))
	}
}

func TestEveryJSONDocumentCarriesAVersion(t *testing.T) {
	reg := registryRepo(t)
	installFromURL(t, reg, "claude")
	for _, args := range [][]string{
		{"skill", "list", "--installed", "--json"},
		{"skill", "list", fileURL(reg), "--json"},
		{"skill", "info", fileURL(reg), "alpha", "--json"},
		{"config", "list", "--json"},
	} {
		stdout, _, err := runSplit(t, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		decode(t, stdout)
	}
}

func TestExitErrorUnwraps(t *testing.T) {
	inner := errors.New("boom")
	if !errors.Is(usageError(inner), inner) {
		t.Error("ExitError should unwrap to its cause")
	}
	if usageError(nil) != nil {
		t.Error("usageError(nil) must be nil")
	}
}
