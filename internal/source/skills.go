package source

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill is one skill found in a checkout.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Owner       string `json:"owner,omitempty"`
	Rel         string `json:"path"` // directory relative to the checkout, "." for the root
}

type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Metadata    struct {
		Owner string `yaml:"owner"`
		Team  string `yaml:"team"`
	} `yaml:"metadata"`
}

// Discover finds skills in dir without needing an index file. It looks for
// skills/<name>/SKILL.md, then <name>/SKILL.md, then a SKILL.md at the root
// (a single-skill repo named fallback).
func Discover(dir, fallback string) ([]Skill, error) {
	for _, base := range []string{"skills", "."} {
		skills, err := scanChildren(dir, base)
		if err != nil {
			return nil, err
		}
		if len(skills) > 0 {
			sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
			return skills, nil
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
		s := readSkill(dir, ".", fallback)
		return []Skill{s}, nil
	}
	return nil, nil
}

func scanChildren(dir, base string) ([]Skill, error) {
	entries, err := os.ReadDir(filepath.Join(dir, base))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filepath.Join(dir, base), err)
	}
	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || e.Name() == "node_modules" {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(base, e.Name()))
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel), "SKILL.md")); err != nil {
			continue
		}
		skills = append(skills, readSkill(dir, rel, e.Name()))
	}
	return skills, nil
}

func readSkill(dir, rel, name string) Skill {
	s := Skill{Name: name, Rel: rel}
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel), "SKILL.md"))
	if err != nil {
		return s
	}
	fm, ok := parseFrontmatter(string(data))
	if !ok {
		return s
	}
	s.Description = strings.TrimSpace(fm.Description)
	s.Owner = fm.Metadata.Owner
	if s.Owner == "" {
		s.Owner = fm.Metadata.Team
	}
	if rel == "." && fm.Name != "" {
		s.Name = fm.Name
	}
	return s
}

func parseFrontmatter(content string) (frontmatter, bool) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return frontmatter{}, false
	}
	rest := content[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return frontmatter{}, false
	}
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return frontmatter{}, false
	}
	return fm, true
}

// Find returns the skill with the given name.
func Find(skills []Skill, name string) (Skill, bool) {
	for _, s := range skills {
		if s.Name == name {
			return s, true
		}
	}
	return Skill{}, false
}

// CopyDir copies src into dst, skipping .git and symlinks (a skill must not
// reach outside its own directory).
func CopyDir(src, dst string) ([]string, error) {
	var written []string
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return fs.SkipDir
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := copyFile(path, target); err != nil {
			return err
		}
		written = append(written, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("copy %s: %w", src, err)
	}
	sort.Strings(written)
	return written, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// normalizeLineEndings turns CRLF into LF in text files so a skill hashes the
// same on every platform (git may check files out with CRLF on Windows).
// Binary files, recognized by a NUL byte, are left untouched.
func normalizeLineEndings(data []byte) []byte {
	if bytes.IndexByte(data, 0) >= 0 {
		return data
	}
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}

// HashDir returns a sha256 per file, keyed by slash-separated relative path.
// Line endings in text files are normalized; see normalizeLineEndings.
func HashDir(dir string) (map[string]string, error) {
	hashes := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(normalizeLineEndings(data))
		hashes[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("hash %s: %w", dir, err)
	}
	return hashes, nil
}

// DiffFiles compares two hash maps.
func DiffFiles(old, cur map[string]string) (added, modified, removed []string) {
	for p, h := range cur {
		oh, ok := old[p]
		switch {
		case !ok:
			added = append(added, p)
		case oh != h:
			modified = append(modified, p)
		}
	}
	for p := range old {
		if _, ok := cur[p]; !ok {
			removed = append(removed, p)
		}
	}
	sort.Strings(added)
	sort.Strings(modified)
	sort.Strings(removed)
	return
}
