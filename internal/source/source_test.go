package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/source"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		kind    source.Kind
		display string
		wantErr bool
	}{
		{"hasankhatib/ai", source.KindGitHub, "hasankhatib/ai", false},
		{"https://github.com/hasankhatib/ai.git", source.KindGitHub, "hasankhatib/ai", false},
		{"git@github.com:hasankhatib/ai.git", source.KindGitHub, "hasankhatib/ai", false},
		{"https://gitlab.com/team/skills.git", source.KindGit, "https://gitlab.com/team/skills.git", false},
		{"git@example.com:team/skills.git", source.KindGit, "git@example.com:team/skills.git", false},
		{"", "", "", true},
		{"justaname", "", "", true},
		{"a/b/c", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			s, err := source.Parse(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
			if err == nil && (s.Kind != tt.kind || s.Display != tt.display) {
				t.Fatalf("got %+v", s)
			}
		})
	}
}

func TestParseLocalPaths(t *testing.T) {
	dir := t.TempDir()
	for _, in := range []string{dir, "/abs/path", "./rel", "../rel", "~/skills"} {
		s, err := source.Parse(in)
		if err != nil || s.Kind != source.KindLocal || s.Path == "" {
			t.Errorf("Parse(%q) = %+v, %v", in, s, err)
		}
	}
}

func TestLooksLikeSource(t *testing.T) {
	for in, want := range map[string]bool{
		"my-skill": false, "My Skill": false, "owner/repo": true, "./x": true,
		"https://h/x.git": true, "git@h:x/y.git": true, "~/skills": true, "..": true,
	} {
		if got := source.LooksLikeSource(in); got != want {
			t.Errorf("LooksLikeSource(%q) = %v", in, got)
		}
	}
}

func put(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("---\nname: x\ndescription: d\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverLayouts(t *testing.T) {
	t.Run("skills directory", func(t *testing.T) {
		d := t.TempDir()
		put(t, filepath.Join(d, "skills", "b", "SKILL.md"))
		put(t, filepath.Join(d, "skills", "a", "SKILL.md"))
		os.MkdirAll(filepath.Join(d, "skills", "notaskill"), 0o755)
		got, _ := source.Discover(d, "repo")
		if len(got) != 2 || got[0].Name != "a" || got[0].Rel != "skills/a" || got[0].Description != "d" {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("top level directories", func(t *testing.T) {
		d := t.TempDir()
		put(t, filepath.Join(d, "one", "SKILL.md"))
		put(t, filepath.Join(d, ".hidden", "SKILL.md"))
		got, _ := source.Discover(d, "repo")
		if len(got) != 1 || got[0].Name != "one" {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("root skill", func(t *testing.T) {
		d := t.TempDir()
		put(t, filepath.Join(d, "SKILL.md"))
		got, _ := source.Discover(d, "repo")
		if len(got) != 1 || got[0].Rel != "." || got[0].Name != "x" {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("empty", func(t *testing.T) {
		if got, _ := source.Discover(t.TempDir(), "repo"); len(got) != 0 {
			t.Fatalf("got %+v", got)
		}
	})
}

func TestCopyDirSkipsGitAndSymlinks(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "out")
	put(t, filepath.Join(src, "SKILL.md"))
	put(t, filepath.Join(src, ".git", "config"))
	if err := os.Symlink("/etc/passwd", filepath.Join(src, "link")); err != nil {
		t.Skip("symlinks unavailable")
	}
	files, err := source.CopyDir(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "SKILL.md" {
		t.Fatalf("files = %v", files)
	}
	if _, err := os.Lstat(filepath.Join(dst, "link")); err == nil {
		t.Fatal("symlink was copied")
	}
}

func TestHashDirIgnoresLineEndingsInTextFilesOnly(t *testing.T) {
	lf, crlf := t.TempDir(), t.TempDir()
	for dir, text := range map[string]string{lf: "a\nb\n", crlf: "a\r\nb\r\n"} {
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "bin.dat"), []byte("x\r\n\x00y"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Differ the binary file in the CRLF dir: it must still count as different.
	if err := os.WriteFile(filepath.Join(crlf, "bin.dat"), []byte("x\n\x00y"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, _ := source.HashDir(lf)
	b, _ := source.HashDir(crlf)
	if a["SKILL.md"] != b["SKILL.md"] {
		t.Fatal("text files that differ only in line endings should hash the same")
	}
	if a["bin.dat"] == b["bin.dat"] {
		t.Fatal("binary files must be hashed byte for byte")
	}
}
