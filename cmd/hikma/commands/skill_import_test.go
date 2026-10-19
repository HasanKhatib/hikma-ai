package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hasankhatib/hikma-ai/internal/lock"
	"github.com/hasankhatib/hikma-ai/internal/preflight"
	"github.com/hasankhatib/hikma-ai/internal/source"
)

const importAlphaMD = "---\nname: alpha\ndescription: Alpha skill\nlicense: MIT\nmetadata:\n  owner: someone\n  github-repo: https://github.com/o/r\n  github-ref: refs/heads/main\n  github-tree-sha: abc123\n---\n# Alpha\nSee [other](../beta/SKILL.md).\n"

// upstreamRepo builds a git repo with two skills: alpha (licensed, installed
// by gh skill once, has a script) and beta (no licence).
func upstreamRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	git(t, dir, "init", "--quiet", "-b", "main")
	write(t, filepath.Join(dir, "skills", "alpha", "SKILL.md"), importAlphaMD)
	write(t, filepath.Join(dir, "skills", "alpha", "scripts", "run.sh"), "#!/bin/sh\necho v1\n")
	if err := os.Chmod(filepath.Join(dir, "skills", "alpha", "scripts", "run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "skills", "beta", "SKILL.md"), "---\nname: beta\ndescription: Beta skill\n---\n# Beta\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "--quiet", "-m", "init")
	return dir
}

// importGH is fakeGH plus the registry listings and blob capture import needs.
type importGH struct {
	*fakeGH
	existing  map[string]string // skill folder -> top-level listing
	oldFiles  string            // files the registry has under the existing skill, newline separated
	blobs     map[string]string // blob sha -> decoded content
	treeBody  map[string]any
	rateLimit int // fail this many calls with a 429 first
}

func newImportGH() *importGH {
	return &importGH{fakeGH: newFakeGH(), existing: map[string]string{}, blobs: map[string]string{}}
}

func (g *importGH) run(stdin string, args ...string) (string, error) {
	if g.rateLimit > 0 {
		g.rateLimit--
		return "", fmt.Errorf("gh api: HTTP 429: too many requests")
	}
	endpoint, jq := "", ""
	for i, a := range args {
		if i == 1 {
			endpoint = a
		}
		if a == "--jq" {
			jq = args[i+1]
		}
	}
	switch {
	case endpoint == "repos/org/reg/contents/skills":
		if len(g.existing) == 0 {
			return "", fmt.Errorf("HTTP 404: Not Found")
		}
		var names []string
		for n := range g.existing {
			names = append(names, n)
		}
		return strings.Join(names, "\n"), nil
	case strings.HasPrefix(endpoint, "repos/org/reg/contents/skills/") && !strings.Contains(endpoint, "?"):
		return g.existing[strings.TrimPrefix(endpoint, "repos/org/reg/contents/skills/")], nil
	case strings.Contains(endpoint, "/git/trees/") && strings.HasSuffix(endpoint, "?recursive=1"):
		if jq == ".truncated" {
			return "false", nil
		}
		return g.oldFiles, nil
	}
	out, err := g.fakeGH.run(stdin, args...)
	if err == nil && strings.HasSuffix(endpoint, "/git/blobs") {
		var body map[string]string
		_ = json.Unmarshal([]byte(stdin), &body)
		data, _ := base64.StdEncoding.DecodeString(body["content"])
		g.blobs[out] = string(data)
	}
	if strings.HasSuffix(endpoint, "/git/trees") && err == nil {
		var body map[string]any
		_ = json.Unmarshal([]byte(stdin), &body)
		g.treeBody = body
	}
	return out, err
}

// treeFiles returns path -> blob content for the last tree that was created.
func (g *importGH) treeFiles(t *testing.T) (files map[string]string, removed []string) {
	t.Helper()
	if g.treeBody == nil {
		t.Fatal("no tree was created")
	}
	files = map[string]string{}
	for _, e := range g.treeBody["tree"].([]any) {
		m := e.(map[string]any)
		if m["sha"] == nil {
			removed = append(removed, m["path"].(string))
			continue
		}
		files[m["path"].(string)] = g.blobs[m["sha"].(string)]
	}
	return files, removed
}

func useImportFake(t *testing.T, g *importGH) {
	t.Helper()
	oldRun, oldSleep, oldNow, oldPre, oldGHSleep, oldNotice := ghRun, forkSleep, pushNow, preflightCheck, ghSleep, ghNotice
	ghRun = g.run
	forkSleep = func(time.Duration) {}
	ghSleep = func(time.Duration) {}
	ghNotice = &strings.Builder{}
	pushNow = func() time.Time { return time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC) }
	preflightCheck = func() preflight.Result { return preflight.Result{} }
	t.Cleanup(func() {
		ghRun, forkSleep, pushNow, preflightCheck, ghSleep, ghNotice = oldRun, oldSleep, oldNow, oldPre, oldGHSleep, oldNotice
	})
}

func noGH(t *testing.T) {
	t.Helper()
	old := ghRun
	ghRun = func(string, ...string) (string, error) {
		t.Fatalf("gh must not be called")
		return "", nil
	}
	t.Cleanup(func() { ghRun = old })
}

func TestImportDryRunReviewsAndTouchesNothing(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	noGH(t)

	out, err := run(t, "skill", "import", up, "alpha", "--dry-run", "--registry", "org/reg")
	if err != nil {
		t.Fatalf("dry run: %v\n%s", err, out)
	}
	for _, want := range []string{
		"Importing alpha", "skills/alpha", "scripts/run.sh", "licence:  MIT (license in SKILL.md)",
		"changed:  SKILL.md: removed metadata.github-repo, metadata.github-ref, metadata.github-tree-sha",
		"scripts and executables:", "links outside the skill folder: ../beta/SKILL.md",
		"Dry run: nothing was written, and the registry was not contacted.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q:\n%s", want, out)
		}
	}
}

func TestImportDryRunJSONIsOneDocument(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	noGH(t)

	stdout, stderr, err := runSplit(t, "skill", "import", up, "alpha", "--dry-run", "--json", "--registry", "org/reg")
	if err != nil {
		t.Fatalf("dry run: %v\n%s", err, stderr)
	}
	doc := decode(t, stdout)
	if doc["dryRun"] != true {
		t.Errorf("dryRun = %v", doc["dryRun"])
	}
	src := doc["source"].(map[string]any)
	if src["source"] != "local" || src["path"] != "skills/alpha" || src["commit"] == nil || src["tree"] == nil {
		t.Errorf("source = %v", src)
	}
	if len(doc["files"].([]any)) != 2 || len(doc["changes"].([]any)) != 1 {
		t.Errorf("files/changes = %v / %v", doc["files"], doc["changes"])
	}
	if doc["target"].(map[string]any)["registry"] != "org/reg" || doc["result"] != nil {
		t.Errorf("target/result = %v / %v", doc["target"], doc["result"])
	}
	if !strings.Contains(stderr, "Importing alpha") {
		t.Errorf("the prose belongs on stderr with --json:\n%s", stderr)
	}
	if strings.Contains(stdout, "Importing alpha") {
		t.Errorf("stdout must be only the document")
	}
}

func TestImportWithPROpensAPullRequest(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	useImportFake(t, g)

	out, err := run(t, "skill", "import", up, "alpha", "--pr", "--yes", "--registry", "org/reg")
	if err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	if !g.has("POST repos/org/reg/pulls") || g.has("PATCH repos/org/reg/git/refs/heads/main") {
		t.Fatalf("want a pull request and no direct commit, calls: %v", g.calls)
	}
	if body := g.bodies["POST repos/org/reg/git/refs"]; body["ref"] != "refs/heads/import/alpha" {
		t.Errorf("branch = %v", body["ref"])
	}
	if msg := g.bodies["POST repos/org/reg/git/commits"]["message"]; msg != "feat(skills): import alpha from local" {
		t.Errorf("commit message = %v", msg)
	}
	pr := g.bodies["POST repos/org/reg/pulls"]
	if !strings.Contains(pr["body"].(string), "Imports the `alpha` skill") || !strings.Contains(pr["body"].(string), "scripts/run.sh") {
		t.Errorf("PR body = %v", pr["body"])
	}
	if !strings.Contains(out, "Imported alpha into a pull request") || !strings.Contains(out, "hikma skill install alpha") {
		t.Errorf("output:\n%s", out)
	}
}

// Import publishes the way push does: straight to the default branch when you
// can write, and the contents are what the review showed.
func TestImportCommitsToTheDefaultBranchWhenYouCanWrite(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	useImportFake(t, g)

	out, err := run(t, "skill", "import", up, "alpha", "--yes", "--registry", "org/reg")
	if err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	if !g.has("PATCH repos/org/reg/git/refs/heads/main") || g.has("POST repos/org/reg/pulls") {
		t.Fatalf("want a direct commit, calls: %v", g.calls)
	}
	if !strings.Contains(out, "committed straight to the default branch, with no review (add --pr for a pull request)") {
		t.Errorf("the confirmation must say there is no review:\n%s", out)
	}
	if !strings.Contains(out, "Imported alpha into org/reg") {
		t.Errorf("output:\n%s", out)
	}

	files, removed := g.treeFiles(t)
	if len(removed) != 0 {
		t.Errorf("nothing should be removed, got %v", removed)
	}
	skillMD := files["skills/alpha/SKILL.md"]
	if strings.Contains(skillMD, "github-") || !strings.Contains(skillMD, "owner: someone") || !strings.Contains(skillMD, "# Alpha") {
		t.Errorf("SKILL.md was not stripped cleanly:\n%s", skillMD)
	}
	if files["skills/alpha/scripts/run.sh"] != "#!/bin/sh\necho v1\n" {
		t.Errorf("run.sh changed: %q", files["skills/alpha/scripts/run.sh"])
	}
	origin, err := source.ParseOrigin([]byte(files["skills/alpha/"+source.OriginFile]))
	if err != nil {
		t.Fatal(err)
	}
	if origin.Version != 1 || origin.Source != "local" || origin.Path != "skills/alpha" || origin.Commit == nil || origin.Tree == nil ||
		origin.License == nil || *origin.License != "MIT" || len(origin.Changes) != 1 || origin.Changes[0].Reason != "strip-github-metadata" {
		t.Errorf("origin = %+v", origin)
	}
	if origin.Files["SKILL.md"] != source.Hash([]byte(skillMD)) || origin.Files["scripts/run.sh"] == "" || origin.Files[source.OriginFile] != "" {
		t.Errorf("origin files = %v", origin.Files)
	}
}

func TestImportFollowsTheRegistryDefaultBranchWhateverItIsCalled(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	g.branch = "master"
	g.refs["org/reg/master"] = "base1"
	useImportFake(t, g)

	if out, err := run(t, "skill", "import", up, "alpha", "--yes", "--registry", "org/reg"); err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	if !g.has("PATCH repos/org/reg/git/refs/heads/master") {
		t.Fatalf("want a commit to master, calls: %v", g.calls)
	}
}

func TestImportOpensAPullRequestWhenTheBranchRequiresOne(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	g.rules = "deletion,pull_request"
	useImportFake(t, g)

	out, err := run(t, "skill", "import", up, "alpha", "--yes", "--registry", "org/reg")
	if err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	if !g.has("POST repos/org/reg/pulls") || g.has("PATCH repos/org/reg/git/refs/heads/main") {
		t.Fatalf("want a pull request, calls: %v", g.calls)
	}
	if !strings.Contains(out, "as a pull request (the default branch requires one)") || !strings.Contains(out, "requires pull requests") {
		t.Errorf("the output must say why:\n%s", out)
	}
}

func TestImportGoesThroughYourForkWithoutWriteAccess(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	g.canPush = "false"
	useImportFake(t, g)

	out, err := run(t, "skill", "import", up, "alpha", "--yes", "--registry", "org/reg")
	if err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	if !g.has("POST repos/org/reg/forks") || !g.has("POST repos/org/reg/pulls") {
		t.Fatalf("want a fork and a pull request, calls: %v", g.calls)
	}
	if !strings.Contains(out, "as a pull request from your fork (you have no write access)") {
		t.Errorf("output:\n%s", out)
	}
}

func TestImportRefusesToOverwriteAndForceReplacesWithoutLeavingStaleFiles(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	g.existing["alpha"] = "SKILL.md, old.md"
	g.oldFiles = "skills/alpha/SKILL.md\nskills/alpha/old.md"
	useImportFake(t, g)

	out, err := run(t, "skill", "import", up, "alpha", "--yes", "--registry", "org/reg")
	if err == nil || !strings.Contains(err.Error(), `already has a skill called "alpha"`) || !strings.Contains(err.Error(), "registry has: SKILL.md, old.md") {
		t.Fatalf("want a refusal with a summary, got %v\n%s", err, out)
	}
	if g.has("POST repos/org/reg/git/blobs") {
		t.Fatal("nothing may be uploaded before the refusal")
	}

	out, err = run(t, "skill", "import", up, "alpha", "--yes", "--force", "--registry", "org/reg")
	if err != nil {
		t.Fatalf("forced import: %v\n%s", err, out)
	}
	_, removed := g.treeFiles(t)
	if len(removed) != 1 || removed[0] != "skills/alpha/old.md" {
		t.Errorf("the stale file must be removed, got %v", removed)
	}
}

func TestImportRefusesACaseOnlyCollisionEvenWithForce(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	g.existing["Alpha"] = "SKILL.md"
	useImportFake(t, g)

	_, err := run(t, "skill", "import", up, "alpha", "--yes", "--force", "--registry", "org/reg")
	if err == nil || !strings.Contains(err.Error(), "letter case") || !strings.Contains(err.Error(), "--as") {
		t.Fatalf("want a letter-case refusal, got %v", err)
	}
}

func TestImportAsRenamesOnlyTheNameLine(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	useImportFake(t, g)

	out, err := run(t, "skill", "import", up, "alpha", "--as", "alpha-copy", "--yes", "--registry", "org/reg")
	if err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	files, _ := g.treeFiles(t)
	md := files["skills/alpha-copy/SKILL.md"]
	if !strings.Contains(md, "name: alpha-copy\n") || !strings.Contains(md, "description: Alpha skill") || strings.Contains(md, "github-") {
		t.Errorf("SKILL.md:\n%s", md)
	}
	origin, _ := source.ParseOrigin([]byte(files["skills/alpha-copy/"+source.OriginFile]))
	var reasons []string
	for _, c := range origin.Changes {
		reasons = append(reasons, c.Reason)
	}
	if strings.Join(reasons, ",") != "strip-github-metadata,rename" {
		t.Errorf("changes = %v", reasons)
	}
}

func TestImportNeverPromptsWithoutATerminal(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	useImportFake(t, g)

	_, err := run(t, "skill", "import", up, "alpha", "--registry", "org/reg")
	if err == nil || !strings.Contains(err.Error(), "confirmation required: re-run with --yes") {
		t.Fatalf("want a confirmation error naming --yes, got %v", err)
	}
	if g.has("POST repos/org/reg/git/blobs") {
		t.Fatal("nothing may be written without confirmation")
	}
}

func TestImportWithoutANameListsTheSkillsAndExitsWithUsageError(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	noGH(t)

	stdout, _, err := runSplit(t, "skill", "import", up, "--registry", "org/reg")
	if ExitCode(err) != ExitUsage {
		t.Fatalf("exit = %d, want 2 (%v)", ExitCode(err), err)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 2 || lines[0] != "alpha\tskills/alpha\tAlpha skill" || lines[1] != "beta\tskills/beta\tBeta skill" {
		t.Fatalf("want one tab-separated line per skill, got %q", stdout)
	}
}

func TestImportNamesTheLicenceProblem(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	g := newImportGH()
	useImportFake(t, g)

	out, err := run(t, "skill", "import", up, "beta", "--yes", "--registry", "org/reg")
	if err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	if strings.Count(out, "no licence found") < 2 {
		t.Errorf("the missing licence must show in the review and the confirmation:\n%s", out)
	}
	origin, _ := source.ParseOrigin([]byte(mustTree(t, g)["skills/beta/"+source.OriginFile]))
	if origin.License != nil {
		t.Errorf("license = %v, want null", *origin.License)
	}
}

func mustTree(t *testing.T, g *importGH) map[string]string {
	files, _ := g.treeFiles(t)
	return files
}

func TestImportFromAnAgentFolderPointsAtPush(t *testing.T) {
	proj := project(t)
	noGH(t)
	write(t, filepath.Join(proj, ".claude", "skills", "mine", "SKILL.md"), "---\nname: mine\ndescription: d\n---\n")

	_, err := run(t, "skill", "import", "./.claude/skills/mine", "--dry-run", "--registry", "org/reg")
	if err == nil || !strings.Contains(err.Error(), "already a local skill") || !strings.Contains(err.Error(), "hikma skill push mine") {
		t.Fatalf("want a redirect to push, got %v", err)
	}
}

func TestImportUsageErrors(t *testing.T) {
	project(t)
	noGH(t)
	for _, tc := range []struct {
		name string
		args []string
		frag string
	}{
		{"no source", []string{"skill", "import"}, "missing argument"},
		{"bare name", []string{"skill", "import", "alpha"}, "is a skill name, not a source"},
		{"json needs yes", []string{"skill", "import", "o/r", "x", "--json"}, "--json never asks a question"},
		{"ref with a tree URL", []string{"skill", "import", "https://github.com/o/r/tree/main/skills/x", "--ref", "v1"}, "drop --ref"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := run(t, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.frag) || ExitCode(err) != ExitUsage {
				t.Fatalf("want a usage error containing %q, got %v (exit %d)", tc.frag, err, ExitCode(err))
			}
		})
	}
	_, err := run(t, "skill", "import", "https://gitlab.com/team/skills.git", "x", "--registry", "org/reg")
	if err == nil || !strings.Contains(err.Error(), "only github.com repositories and local folders") {
		t.Fatalf("want a non-GitHub refusal, got %v", err)
	}
}

// A GitHub source is read from git objects. The test points github.com at a
// local repository with git's own URL rewriting, so no network is used.
func TestImportReadsAGitHubSourceFromGitObjects(t *testing.T) {
	up := upstreamRepo(t)
	project(t)
	noGH(t)
	url := "file://" + filepath.ToSlash(up)
	if runtime.GOOS == "windows" {
		url = "file:///" + filepath.ToSlash(up)
	}
	t.Setenv("GIT_CONFIG_COUNT", "2")
	t.Setenv("GIT_CONFIG_KEY_0", "url."+url+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://github.com/owner/repo.git")
	t.Setenv("GIT_CONFIG_KEY_1", "core.autocrlf")
	t.Setenv("GIT_CONFIG_VALUE_1", "true")

	for _, arg := range []string{
		"owner/repo/skills/alpha",
		"https://github.com/owner/repo/tree/main/skills/alpha",
		"https://github.com/owner/repo/blob/main/skills/alpha/SKILL.md",
	} {
		stdout, stderr, err := runSplit(t, "skill", "import", arg, "--dry-run", "--json", "--registry", "org/reg")
		if err != nil {
			t.Fatalf("%s: %v\n%s", arg, err, stderr)
		}
		src := decode(t, stdout)["source"].(map[string]any)
		if src["source"] != "owner/repo" || src["path"] != "skills/alpha" || src["commit"] == nil || src["tree"] == nil {
			t.Errorf("%s: source = %v", arg, src)
		}
		if !strings.HasPrefix(src["readFrom"].(string), "git objects at ") {
			t.Errorf("%s: readFrom = %v", arg, src["readFrom"])
		}
	}
	// SKILL.md bytes are the stored ones, so they still have plain \n line endings.
	stdout, _, err := runSplit(t, "skill", "import", "owner/repo", "beta", "--dry-run", "--json", "--registry", "org/reg")
	if err != nil {
		t.Fatal(err)
	}
	files := decode(t, stdout)["files"].([]any)
	if size := files[0].(map[string]any)["size"].(float64); size != float64(len("---\nname: beta\ndescription: Beta skill\n---\n# Beta\n")) {
		t.Errorf("SKILL.md size = %v, line endings were rewritten", size)
	}
}

func TestPushRetriesWhenGitHubRateLimits(t *testing.T) {
	g := newImportGH()
	g.rateLimit = 2
	useImportFake(t, g)
	sleeps := 0
	ghSleep = func(d time.Duration) { sleeps++ }

	if _, err := ghGet("repos/org/reg", ".default_branch"); err != nil {
		t.Fatalf("want success after the retries, got %v", err)
	}
	if sleeps != 2 {
		t.Errorf("slept %d times, want 2", sleeps)
	}
	if !strings.Contains(ghNotice.(*strings.Builder).String(), "rate limiting requests; waiting 1m0s before retrying (1 of 3)") {
		t.Errorf("notice = %q", ghNotice.(*strings.Builder).String())
	}

	g.rateLimit = 99
	if _, err := ghGet("repos/org/reg", ".default_branch"); err == nil || !strings.Contains(err.Error(), "still rate limiting requests after 3 retries") {
		t.Fatalf("want a clear failure, got %v", err)
	}

	// Other errors are not retried.
	g.rateLimit = 0
	sleeps = 0
	if _, err := ghGet("repos/org/reg/git/ref/heads/nope", ".object.sha"); err == nil || sleeps != 0 {
		t.Fatalf("a 404 must fail at once, got %v after %d sleeps", err, sleeps)
	}
}

// install takes the same folder and URL forms, through the shared parser.
func TestInstallTakesAFolderPathAndATreeURL(t *testing.T) {
	up := upstreamRepo(t)
	proj := project(t)
	url := "file://" + filepath.ToSlash(up)
	if runtime.GOOS == "windows" {
		url = "file:///" + filepath.ToSlash(up)
	}
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url."+url+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", "https://github.com/owner/repo.git")

	for _, arg := range []string{"owner/repo/skills/alpha", "https://github.com/owner/repo/tree/main/skills/alpha"} {
		if out, err := run(t, "skill", "install", arg, "--agent", "claude", "--force"); err != nil {
			t.Fatalf("install %s: %v\n%s", arg, err, out)
		}
		if _, err := os.Stat(filepath.Join(proj, ".claude", "skills", "alpha", "SKILL.md")); err != nil {
			t.Fatalf("%s: skill not installed: %v", arg, err)
		}
	}
	lf, err := lock.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	if e := lf.Skills[".claude/skills/alpha"]; e.Source != "owner/repo" || e.Ref != "main" && e.Ref != "" {
		t.Errorf("lock entry = %+v: the source must be the repository, not the URL", e)
	}

	out, err := run(t, "skill", "install", "owner/repo/skills/nope", "--agent", "claude")
	if err == nil || !strings.Contains(err.Error(), "no skill to install at skills/nope") || !strings.Contains(err.Error(), "hikma skill import") {
		t.Fatalf("want a pointer to import, got %v\n%s", err, out)
	}
}

// A registry that holds an imported skill still validates, and installing the
// skill carries the origin file along like any other file in the folder.
func TestRegistryWithAnImportedSkillValidatesAndInstalls(t *testing.T) {
	reg := t.TempDir()
	write(t, filepath.Join(reg, "skills", "x", "SKILL.md"), "---\nname: x\ndescription: d\n---\n# X\n")
	write(t, filepath.Join(reg, "skills", "x", source.OriginFile), `{"version": 1, "source": "o/r", "ref": null, "commit": null, "tree": null, "path": "skills/x", "license": null, "files": {}, "changes": []}`+"\n")
	proj := project(t)

	if out, err := run(t, "registry", "validate", reg); err != nil {
		t.Fatalf("validate: %v\n%s", err, out)
	}
	if out, err := run(t, "skill", "install", reg, "x", "--agent", "claude"); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "skills", "x", source.OriginFile)); err != nil {
		t.Fatalf("the origin file should be installed with the skill: %v", err)
	}
}
