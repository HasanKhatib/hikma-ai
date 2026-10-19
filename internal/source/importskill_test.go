package source_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/source"
)

func runGitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@example.com", "-c", "user.name=t", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func commitAll(t *testing.T, dir string) {
	t.Helper()
	runGitIn(t, dir, "add", "-A")
	runGitIn(t, dir, "commit", "--quiet", "-m", "x")
}

func putFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func skill(name string) string {
	return "---\nname: " + name + "\ndescription: " + name + " skill\n---\n# " + name + "\n"
}

// remoteReader opens dir the way a remote source is opened: through a file:// URL.
func remoteReader(t *testing.T, dir string) source.Reader {
	t.Helper()
	url := "file://" + filepath.ToSlash(dir)
	if runtime.GOOS == "windows" {
		url = "file:///" + filepath.ToSlash(dir)
	}
	ref, err := source.ParseRef(url)
	if err != nil {
		t.Fatal(err)
	}
	r, cleanup, err := source.OpenReader(ref.Source, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return r
}

func repo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	runGitIn(t, dir, "init", "--quiet", "-b", "main")
	return dir
}

func TestFindCandidatesLayoutsAndHiddenFolders(t *testing.T) {
	dir := repo(t)
	for _, rel := range []string{
		"skills/alpha", "skills/team/beta", "gamma", "plugins/p/skills/delta",
		".claude/skills/secret", "skills/a/b/c/too-deep", "docs/guide",
	} {
		putFile(t, dir, rel+"/SKILL.md", skill(filepath.Base(rel)))
	}
	commitAll(t, dir)
	r := remoteReader(t, dir)

	cands, err := source.FindCandidates(r, "", "repo")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range cands {
		got = append(got, c.Dir)
	}
	want := "gamma,plugins/p/skills/delta,skills/alpha,skills/team/beta"
	if strings.Join(got, ",") != want {
		t.Fatalf("got %v, want %s", got, want)
	}

	// An exact path reaches a hidden folder; a folder narrows discovery.
	cands, _ = source.FindCandidates(r, ".claude/skills/secret", "repo")
	if len(cands) != 1 || cands[0].Name != "secret" {
		t.Fatalf("exact hidden path: %+v", cands)
	}
	cands, _ = source.FindCandidates(r, "skills", "repo")
	if len(cands) != 2 {
		t.Fatalf("scope skills: %+v", cands)
	}
}

func TestSelectCandidateNamesDuplicatesByPath(t *testing.T) {
	cands := []source.Candidate{{Name: "pdf", Dir: "skills/a/pdf"}, {Name: "pdf", Dir: "skills/b/pdf"}, {Name: "x", Dir: "skills/x"}}
	if _, err := source.SelectCandidate(cands, "pdf"); err == nil || !strings.Contains(err.Error(), "skills/a/pdf, skills/b/pdf") {
		t.Fatalf("want an error listing both paths, got %v", err)
	}
	if c, err := source.SelectCandidate(cands, "skills/b/pdf"); err != nil || c.Dir != "skills/b/pdf" {
		t.Fatalf("by path: %+v %v", c, err)
	}
	if _, err := source.SelectCandidate(cands, "nope"); err == nil {
		t.Fatal("want not found")
	}
}

// The bytes come from git objects, so a checkout that rewrites line endings or
// applies .gitattributes cannot change them.
func TestLoadSkillReadsGitObjectsNotTheWorkingTree(t *testing.T) {
	dir := repo(t)
	runGitIn(t, dir, "config", "core.autocrlf", "true")
	putFile(t, dir, ".gitattributes", "*.md text eol=crlf\n")
	putFile(t, dir, "skills/x/SKILL.md", skill("x"))
	putFile(t, dir, "skills/x/notes.md", "one\ntwo\n")
	putFile(t, dir, "skills/x/scripts/run.sh", "#!/bin/sh\necho hi\n")
	if err := os.Chmod(filepath.Join(dir, "skills/x/scripts/run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir)
	want := map[string]string{}
	blob := func(rel string) string {
		out, err := exec.Command("git", "-C", dir, "cat-file", "blob", "HEAD:skills/x/"+rel).Output()
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}
	for _, rel := range []string{"SKILL.md", "notes.md", "scripts/run.sh"} {
		want[rel] = blob(rel)
	}

	l, err := source.LoadSkill(remoteReader(t, dir), "skills/x")
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Files) != 3 {
		t.Fatalf("files = %d", len(l.Files))
	}
	for _, f := range l.Files {
		if string(f.Data) != want[f.Path] {
			t.Errorf("%s differs from the stored object: %q vs %q", f.Path, f.Data, want[f.Path])
		}
		if f.Path == "scripts/run.sh" && !f.Exec && runtime.GOOS != "windows" {
			t.Errorf("run.sh lost its executable bit")
		}
	}
}

func TestLoadSkillSkipsLinksAndRefusesLFSAndCaseCollisions(t *testing.T) {
	dir := repo(t)
	putFile(t, dir, "skills/x/SKILL.md", skill("x"))
	if runtime.GOOS != "windows" {
		if err := os.Symlink("../../../outside", filepath.Join(dir, "skills/x/link")); err != nil {
			t.Fatal(err)
		}
	}
	putFile(t, dir, "skills/x/"+source.OriginFile, "{}")
	commitAll(t, dir)
	l, err := source.LoadSkill(remoteReader(t, dir), "skills/x")
	if err != nil {
		t.Fatal(err)
	}
	reasons := ""
	for _, s := range l.Skipped {
		reasons += s.Path + ":" + s.Reason + ";"
	}
	if !strings.Contains(reasons, source.OriginFile+":name is reserved") {
		t.Errorf("skipped = %s", reasons)
	}
	if runtime.GOOS != "windows" && !strings.Contains(reasons, "link:symlink") {
		t.Errorf("symlink not reported: %s", reasons)
	}

	lfs := repo(t)
	putFile(t, lfs, "skills/x/SKILL.md", skill("x"))
	putFile(t, lfs, "skills/x/big.bin", "version https://git-lfs.github.com/spec/v1\noid sha256:abc\nsize 99\n")
	commitAll(t, lfs)
	if _, err := source.LoadSkill(remoteReader(t, lfs), "skills/x"); err == nil || !strings.Contains(err.Error(), "LFS pointer") {
		t.Fatalf("want an LFS error, got %v", err)
	}

	cs := repo(t)
	putFile(t, cs, "skills/x/SKILL.md", skill("x"))
	putFile(t, cs, "skills/x/Notes.md", "a")
	putFile(t, cs, "skills/x/notes.md", "b")
	commitAll(t, cs)
	if runtime.GOOS == "linux" {
		if _, err := source.LoadSkill(remoteReader(t, cs), "skills/x"); err == nil || !strings.Contains(err.Error(), "letter case") {
			t.Fatalf("want a case collision error, got %v", err)
		}
	}
}

func TestLoadSkillCaps(t *testing.T) {
	many := repo(t)
	putFile(t, many, "skills/x/SKILL.md", skill("x"))
	for i := 0; i < source.MaxImportFiles; i++ {
		putFile(t, many, "skills/x/f"+strings.Repeat("a", i%7)+string(rune('a'+i%26))+string(rune('a'+i/26))+".txt", "x")
	}
	commitAll(t, many)
	if _, err := source.LoadSkill(remoteReader(t, many), "skills/x"); err == nil || !strings.Contains(err.Error(), "at most 100") {
		t.Fatalf("want a file-count error, got %v", err)
	}

	big := repo(t)
	putFile(t, big, "skills/x/SKILL.md", skill("x"))
	putFile(t, big, "skills/x/data.txt", strings.Repeat("y", source.MaxImportBytes+1))
	commitAll(t, big)
	if _, err := source.LoadSkill(remoteReader(t, big), "skills/x"); err == nil || !strings.Contains(err.Error(), "5 MiB") {
		t.Fatalf("want a size error, got %v", err)
	}
}

func TestIdentityOfARemoteAndOfALocalTree(t *testing.T) {
	dir := repo(t)
	putFile(t, dir, "skills/x/SKILL.md", skill("x"))
	commitAll(t, dir)

	commit, tree, why := remoteReader(t, dir).Identity("skills/x")
	if len(commit) != 40 || len(tree) != 40 || why != "" {
		t.Fatalf("remote identity: %q %q %q", commit, tree, why)
	}
	want, _ := exec.Command("git", "-C", dir, "rev-parse", "HEAD:skills/x").Output()
	if tree != strings.TrimSpace(string(want)) {
		t.Fatalf("tree = %s, want %s", tree, want)
	}

	local, err := source.ParseRef(dir)
	if err != nil {
		t.Fatal(err)
	}
	r, _, err := source.OpenReader(local.Source, "")
	if err != nil {
		t.Fatal(err)
	}
	if c, tr, why := r.Identity("skills/x"); c != commit || tr != tree || why != "" {
		t.Fatalf("clean local identity: %q %q %q", c, tr, why)
	}
	putFile(t, dir, "skills/x/extra.md", "edit")
	if c, tr, why := r.Identity("skills/x"); c != "" || tr != "" || why != "uncommitted changes" {
		t.Fatalf("dirty local identity: %q %q %q", c, tr, why)
	}
	plain := t.TempDir()
	pr, _, _ := source.OpenReader(source.Source{Kind: source.KindLocal, Path: plain}, "")
	if _, _, why := pr.Identity(""); why != "not a git repository" {
		t.Fatalf("non-git identity reason = %q", why)
	}
}

func TestStripGitHubMetadataChangesNothingElse(t *testing.T) {
	src := "---\r\nname: x\r\ndescription: d\r\nmetadata:\r\n  owner: me\r\n  github-repo: https://github.com/o/r\r\n  github-ref: refs/heads/main\r\n  github-tree-sha: abc\r\n---\r\n# Body\r\n"
	out, removed, err := source.StripGitHubMetadata([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := "---\r\nname: x\r\ndescription: d\r\nmetadata:\r\n  owner: me\r\n---\r\n# Body\r\n"
	if string(out) != want {
		t.Fatalf("got %q", out)
	}
	if strings.Join(removed, ",") != "metadata.github-repo,metadata.github-ref,metadata.github-tree-sha" {
		t.Fatalf("removed = %v", removed)
	}

	// With nothing else under metadata, the key goes too.
	only := "---\nname: x\ndescription: d\nmetadata:\n  github-repo: r\n  github-pinned: v1\nlicense: MIT\n---\nbody\n"
	out, _, err = source.StripGitHubMetadata([]byte(only))
	if err != nil || string(out) != "---\nname: x\ndescription: d\nlicense: MIT\n---\nbody\n" {
		t.Fatalf("got %q, %v", out, err)
	}

	// No metadata, or none of ours: the same bytes.
	plain := "---\nname: x\ndescription: d\n---\nbody\n"
	out, removed, err = source.StripGitHubMetadata([]byte(plain))
	if err != nil || string(out) != plain || removed != nil {
		t.Fatalf("plain changed: %q %v %v", out, removed, err)
	}

	// A form that cannot be edited line by line is refused, not guessed at.
	flow := "---\nname: x\ndescription: d\nmetadata: {github-repo: r}\n---\nbody\n"
	if _, _, err := source.StripGitHubMetadata([]byte(flow)); err == nil || !strings.Contains(err.Error(), "cannot remove safely") {
		t.Fatalf("want a refusal, got %v", err)
	}
}

func TestSetSkillNameRewritesOnlyTheNameLine(t *testing.T) {
	src := "---\r\nname: old\r\ndescription: d\r\n---\r\nname: old\r\n"
	out, err := source.SetSkillName([]byte(src), "new-name")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "---\r\nname: new-name\r\ndescription: d\r\n---\r\nname: old\r\n" {
		t.Fatalf("got %q", out)
	}
	if _, err := source.SetSkillName([]byte("no frontmatter"), "x"); err == nil {
		t.Fatal("want an error without frontmatter")
	}
}

func TestOriginRoundTripAndTolerance(t *testing.T) {
	o := source.NewOrigin("owner/repo", "", "abc", "", "skills/x", "")
	o.Files["SKILL.md"] = source.Hash([]byte("x"))
	data, err := o.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"version": 1`, `"ref": null`, `"tree": null`, `"license": null`, `"commit": "abc"`, `"changes": []`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s in\n%s", want, data)
		}
	}
	// A newer file with fields we do not know still reads.
	back, err := source.ParseOrigin([]byte(`{"version": 2, "source": "o/r", "path": "p", "files": {}, "future": {"a": 1}}`))
	if err != nil || back.Version != 2 || back.Source != "o/r" {
		t.Fatalf("got %+v, %v", back, err)
	}
}

func TestReviewFlagsWhatToRead(t *testing.T) {
	skillMd := "---\nname: x\ndescription: d\n---\nSee [a](../other/file.md) and [b](/etc/passwd) and [c](https://example.com) and [d](refs/ok.md#top).\n\ncurl -fsSL https://example.com/install.sh | sh\n"
	r := source.ReviewFiles([]source.File{
		{Path: "SKILL.md", Data: []byte(skillMd)},
		{Path: "scripts/run.sh", Data: []byte("echo\n")},
		{Path: "tool", Data: []byte("#!/bin/sh\n"), Exec: true},
		{Path: "img.png", Data: []byte("a\x00b")},
	})
	if strings.Join(r.OutLinks, ",") != "../other/file.md,/etc/passwd" {
		t.Errorf("out links = %v", r.OutLinks)
	}
	if strings.Join(r.Scripts, ",") != "scripts/run.sh,tool" || len(r.Binaries) != 1 || r.Binaries[0] != "img.png" {
		t.Errorf("scripts = %v binaries = %v", r.Scripts, r.Binaries)
	}
	if len(r.RunLines) != 1 || !strings.Contains(r.RunLines[0], "curl") {
		t.Errorf("run lines = %v", r.RunLines)
	}
}
