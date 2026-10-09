package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hasankhatib/hikma-ai/internal/config"
)

// fakeGH is an in-memory stand-in for the parts of the GitHub API push uses.
type fakeGH struct {
	canPush   string            // "true" or "false"
	refs      map[string]string // "owner/repo/branch" -> sha
	openPRs   map[string]string // "head owner:branch" -> url
	sameTree  bool              // make new trees equal the base tree
	hasSkill  bool              // skill already exists upstream
	calls     []string
	bodies    map[string]map[string]any // "METHOD endpoint" -> last body
	blobCount int
}

func newFakeGH() *fakeGH {
	return &fakeGH{
		canPush: "true",
		refs:    map[string]string{"org/reg/main": "base1", "me/reg/main": "forkbase1"},
		openPRs: map[string]string{},
		bodies:  map[string]map[string]any{},
	}
}

func (f *fakeGH) run(stdin string, args ...string) (string, error) {
	if len(args) < 2 || args[0] != "api" {
		return "", fmt.Errorf("unexpected gh call: %v", args)
	}
	endpoint, method, jq := args[1], "GET", ""
	for i, a := range args {
		switch a {
		case "--method":
			method = args[i+1]
		case "--jq":
			jq = args[i+1]
		}
	}
	var body map[string]any
	if stdin != "" {
		_ = json.Unmarshal([]byte(stdin), &body)
	}
	f.calls = append(f.calls, method+" "+endpoint)
	f.bodies[method+" "+endpoint] = body
	notFound := fmt.Errorf("HTTP 404: Not Found")

	switch {
	case method == "GET" && endpoint == "repos/org/reg":
		if jq == ".default_branch" {
			return "main", nil
		}
		return f.canPush, nil
	case method == "POST" && endpoint == "repos/org/reg/forks":
		return "me/reg", nil
	case endpoint == "repos/me/reg" && method == "GET":
		return "me/reg", nil
	case strings.HasSuffix(endpoint, "/merge-upstream"):
		return "", nil
	case strings.Contains(endpoint, "/git/ref/heads/"):
		repo, branch, _ := strings.Cut(strings.TrimPrefix(endpoint, "repos/"), "/git/ref/heads/")
		if sha, ok := f.refs[repo+"/"+branch]; ok {
			return sha, nil
		}
		return "", notFound
	case strings.Contains(endpoint, "/git/commits/") && method == "GET":
		return "tree-" + endpoint[strings.LastIndex(endpoint, "/")+1:], nil
	case strings.HasSuffix(endpoint, "/git/blobs"):
		f.blobCount++
		return fmt.Sprintf("blob-%d", f.blobCount), nil
	case strings.HasSuffix(endpoint, "/git/trees"):
		if f.sameTree {
			return body["base_tree"].(string), nil
		}
		return "newtree", nil
	case strings.HasSuffix(endpoint, "/git/commits") && method == "POST":
		return "newcommit", nil
	case method == "PATCH" && strings.Contains(endpoint, "/git/refs/heads/"):
		return "", nil
	case method == "POST" && strings.HasSuffix(endpoint, "/git/refs"):
		return "", nil
	case strings.Contains(endpoint, "/pulls?head="):
		head := endpoint[strings.Index(endpoint, "head=")+5 : strings.Index(endpoint, "&state")]
		return f.openPRs[head], nil
	case method == "POST" && endpoint == "repos/org/reg/pulls":
		return "https://github.com/org/reg/pull/7", nil
	case strings.Contains(endpoint, "/contents/skills/"):
		if f.hasSkill {
			return "sha", nil
		}
		return "", notFound
	}
	return "", fmt.Errorf("fakeGH: unhandled %s %s", method, endpoint)
}

func (f *fakeGH) has(call string) bool {
	for _, c := range f.calls {
		if c == call {
			return true
		}
	}
	return false
}

func usePushFake(t *testing.T, f *fakeGH) {
	t.Helper()
	oldRun, oldSleep, oldNow := ghRun, forkSleep, pushNow
	ghRun = f.run
	forkSleep = func(time.Duration) {}
	pushNow = func() time.Time { return time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { ghRun, forkSleep, pushNow = oldRun, oldSleep, oldNow })
}

func testPlan() pushPlan {
	return pushPlan{
		upstream: config.Registry{Owner: "org", Repo: "reg"},
		name:     "x",
		files: []pushFile{
			{path: "skills/x/SKILL.md", content: []byte("---\nname: x\ndescription: d\n---\n")},
			{path: "skills/x/scripts/run.sh", content: []byte("echo\n"), exec: true},
		},
	}
}

func TestPushWithWriteAccessCreatesBranchAndPR(t *testing.T) {
	f := newFakeGH()
	usePushFake(t, f)
	var out bytes.Buffer
	pr, err := pushSkill(&out, testPlan())
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if pr != "https://github.com/org/reg/pull/7" {
		t.Fatalf("pr = %q", pr)
	}
	if f.has("POST repos/org/reg/forks") {
		t.Fatal("must not fork when there is write access")
	}
	ref := f.bodies["POST repos/org/reg/git/refs"]
	if ref["ref"] != "refs/heads/skill/x" {
		t.Fatalf("ref = %v", ref)
	}
	commit := f.bodies["POST repos/org/reg/git/commits"]
	if parents := commit["parents"].([]any); parents[0] != "base1" {
		t.Fatalf("commit parents = %v", commit["parents"])
	}
	if commit["message"] != "feat(skills): add x" {
		t.Fatalf("message = %v", commit["message"])
	}
	tree := f.bodies["POST repos/org/reg/git/trees"]["tree"].([]any)
	modes := map[string]string{}
	for _, e := range tree {
		m := e.(map[string]any)
		modes[m["path"].(string)] = m["mode"].(string)
	}
	if modes["skills/x/scripts/run.sh"] != "100755" || modes["skills/x/SKILL.md"] != "100644" {
		t.Fatalf("modes = %v", modes)
	}
	pull := f.bodies["POST repos/org/reg/pulls"]
	if pull["head"] != "skill/x" || pull["base"] != "main" {
		t.Fatalf("pull = %v", pull)
	}
}

func TestPushWithoutWriteAccessGoesThroughFork(t *testing.T) {
	f := newFakeGH()
	f.canPush = "false"
	usePushFake(t, f)
	var out bytes.Buffer
	if _, err := pushSkill(&out, testPlan()); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !f.has("POST repos/org/reg/forks") || !f.has("POST repos/me/reg/merge-upstream") {
		t.Fatalf("expected fork and sync, calls = %v", f.calls)
	}
	if f.has("POST repos/org/reg/git/refs") || f.has("POST repos/org/reg/git/commits") {
		t.Fatal("must not write to upstream without access")
	}
	if !f.has("POST repos/me/reg/git/refs") {
		t.Fatalf("expected branch on the fork, calls = %v", f.calls)
	}
	pull := f.bodies["POST repos/org/reg/pulls"]
	if pull["head"] != "me:skill/x" {
		t.Fatalf("pull head = %v", pull["head"])
	}
}

func TestPushWithOpenPRAddsCommitWithoutForce(t *testing.T) {
	f := newFakeGH()
	f.refs["org/reg/skill/x"] = "branchtip"
	f.openPRs["org:skill/x"] = "https://github.com/org/reg/pull/3"
	f.hasSkill = true
	usePushFake(t, f)
	var out bytes.Buffer
	pr, err := pushSkill(&out, testPlan())
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if pr != "https://github.com/org/reg/pull/3" || f.has("POST repos/org/reg/pulls") {
		t.Fatalf("should reuse the open PR, pr=%q calls=%v", pr, f.calls)
	}
	patch := f.bodies["PATCH repos/org/reg/git/refs/heads/skill/x"]
	if patch == nil || patch["force"] != false || patch["sha"] != "newcommit" {
		t.Fatalf("patch = %v", patch)
	}
	commit := f.bodies["POST repos/org/reg/git/commits"]
	if parents := commit["parents"].([]any); parents[0] != "branchtip" {
		t.Fatalf("new commit should sit on the PR branch tip, parents = %v", commit["parents"])
	}
	if commit["message"] != "feat(skills): update x" {
		t.Fatalf("message = %v", commit["message"])
	}
}

func TestPushLeftoverBranchWithoutPRGetsNewBranch(t *testing.T) {
	f := newFakeGH()
	f.refs["org/reg/skill/x"] = "oldtip" // merged earlier, no open PR
	usePushFake(t, f)
	var out bytes.Buffer
	if _, err := pushSkill(&out, testPlan()); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if f.has("PATCH repos/org/reg/git/refs/heads/skill/x") {
		t.Fatal("must not touch the old branch")
	}
	ref := f.bodies["POST repos/org/reg/git/refs"]
	if ref["ref"] != "refs/heads/skill/x-20261009-120000" {
		t.Fatalf("ref = %v", ref)
	}
	if parents := f.bodies["POST repos/org/reg/git/commits"]["parents"].([]any); parents[0] != "base1" {
		t.Fatalf("new branch should start from the default branch, parents = %v", parents)
	}
}

func TestPushWithNoChangesDoesNothing(t *testing.T) {
	f := newFakeGH()
	f.sameTree = true
	usePushFake(t, f)
	var out bytes.Buffer
	if _, err := pushSkill(&out, testPlan()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No changes to push") || f.has("POST repos/org/reg/git/commits") {
		t.Fatalf("out=%s calls=%v", out.String(), f.calls)
	}
}

func TestPushCodeownersIsOptIn(t *testing.T) {
	f := newFakeGH()
	usePushFake(t, f)
	var out bytes.Buffer
	if _, err := pushSkill(&out, testPlan()); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.calls {
		if strings.Contains(c, "CODEOWNERS") {
			t.Fatalf("CODEOWNERS touched without opt-in: %s", c)
		}
	}
}

func TestReadPushFilesSkipsGitAndKeepsExecBit(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "SKILL.md"), "x")
	write(t, filepath.Join(dir, ".git", "config"), "x")
	if err := os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	files, err := readPushFiles(dir, "x")
	if err != nil || len(files) != 2 {
		t.Fatalf("%v %v", files, err)
	}
	for _, f := range files {
		if strings.Contains(f.path, ".git") {
			t.Fatalf("included %s", f.path)
		}
	}
}
