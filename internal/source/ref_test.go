package source_test

import (
	"strings"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/source"
)

func TestParseRef(t *testing.T) {
	tests := []struct {
		in           string
		display      string
		path, tail   string
		blob         bool
		wantErrFrag  string
		wantKindGit  bool
		wantNoSecret string
	}{
		{in: "owner/repo", display: "owner/repo"},
		{in: "owner/repo/skills/pdf", display: "owner/repo", path: "skills/pdf"},
		{in: "owner/repo/skills/pdf/SKILL.md", display: "owner/repo", path: "skills/pdf"},
		{in: "https://github.com/owner/repo", display: "owner/repo"},
		{in: "https://github.com/owner/repo.git", display: "owner/repo"},
		{in: "https://www.github.com/owner/repo/", display: "owner/repo"},
		{in: "https://github.com/owner/repo/tree/main", display: "owner/repo", tail: "main"},
		{in: "https://github.com/owner/repo/tree/main/skills/pdf?tab=readme#top", display: "owner/repo", tail: "main/skills/pdf"},
		{in: "https://github.com/owner/repo/blob/v1.2/skills/pdf/SKILL.md", display: "owner/repo", tail: "v1.2/skills/pdf/SKILL.md", blob: true},
		{in: "git@github.com:owner/repo.git", display: "owner/repo"},
		{in: "ssh://git@github.com/owner/repo.git", display: "owner/repo"},
		{in: "https://ghp_secret123:x-oauth-basic@github.com/owner/repo", display: "owner/repo", wantNoSecret: "ghp_secret123"},
		{in: "https://gitlab.com/team/skills.git", display: "https://gitlab.com/team/skills.git", wantKindGit: true},
		{in: "https://user:tok@gitlab.com/team/skills.git", display: "https://gitlab.com/team/skills.git", wantKindGit: true, wantNoSecret: "tok"},
		{in: "https://gitlab.com/team/skills/-/tree/main/x", wantErrFrag: "only github.com URLs can name a folder"},
		{in: "https://github.com/owner/repo/issues/4", wantErrFrag: "does not point at a repository"},
		{in: "https://github.com/owner", wantErrFrag: "expected owner/repo"},
		{in: "", wantErrFrag: "empty"},
		{in: "justaname", wantErrFrag: "owner/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			r, err := source.ParseRef(tt.in)
			if tt.wantErrFrag != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrFrag) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErrFrag)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if r.Source.Display != tt.display || r.Path != tt.path || r.Tail != tt.tail || r.Blob != tt.blob {
				t.Fatalf("got display=%q path=%q tail=%q blob=%v", r.Source.Display, r.Path, r.Tail, r.Blob)
			}
			if (r.Source.Kind == source.KindGit) != tt.wantKindGit {
				t.Fatalf("kind = %s", r.Source.Kind)
			}
			if tt.wantNoSecret != "" && (strings.Contains(r.Source.Display, tt.wantNoSecret) || strings.Contains(r.Source.Raw, tt.wantNoSecret) && !tt.wantKindGit) {
				t.Fatalf("credential leaked: %+v", r.Source)
			}
		})
	}
}

func TestParseRefLocalPath(t *testing.T) {
	r, err := source.ParseRef("./some/dir")
	if err != nil || r.Source.Kind != source.KindLocal || r.Path != "" {
		t.Fatalf("got %+v, %v", r, err)
	}
}

func TestResolveTailTakesTheLongestKnownRef(t *testing.T) {
	refs := []string{"main", "feature/skills", "feature", "v1.0"}
	for _, tt := range []struct {
		tail, ref, folder string
		blob, guessed     bool
	}{
		{"main", "main", "", false, false},
		{"main/skills/pdf", "main", "skills/pdf", false, false},
		{"feature/skills/skills/pdf", "feature/skills", "skills/pdf", false, false},
		{"feature/other/pdf", "feature", "other/pdf", false, false},
		{"v1.0/skills/pdf/SKILL.md", "v1.0", "skills/pdf", true, false},
		{"0123abc/skills/pdf", "0123abc", "skills/pdf", false, true},
	} {
		r := source.SkillRef{Tail: tt.tail, Blob: tt.blob}
		ref, folder, guessed := r.ResolveTail(refs)
		if ref != tt.ref || folder != tt.folder || guessed != tt.guessed {
			t.Errorf("%s: got ref=%q folder=%q guessed=%v", tt.tail, ref, folder, guessed)
		}
	}
}

func TestStripCredentials(t *testing.T) {
	for in, want := range map[string]string{
		"https://user:tok@host.example/a/b?x=1#f": "https://host.example/a/b",
		"git@github.com:o/r.git":                  "git@github.com:o/r.git",
		"./local":                                 "./local",
	} {
		if got := source.StripCredentials(in); got != want {
			t.Errorf("%s -> %s, want %s", in, got, want)
		}
	}
}
