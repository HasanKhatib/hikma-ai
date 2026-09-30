// Package registry fetches and caches skill metadata from a user-configured GitHub registry.
package registry

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hasankhatib/hikma-ai/internal/config"
)

const cacheTTL = time.Hour

type Skill struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Owner         string `json:"owner"`
	Email         string `json:"email"`
	LastValidated string `json:"last_validated"`
	Compatibility string `json:"compatibility"`
}

type index struct {
	Generated string  `json:"generated"`
	Skills    []Skill `json:"skills"`
}

type contentEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

func ActiveRegistry() (config.Registry, error) {
	return config.ResolveRegistry("")
}

func FetchIndex() ([]Skill, error) {
	r, err := ActiveRegistry()
	if err != nil {
		return nil, err
	}
	return FetchIndexFrom(r)
}

func FetchIndexFrom(r config.Registry) ([]Skill, error) {
	out, err := exec.Command("gh", "api", r.GitHubAPIRepo()+"/contents/skills/index.json", "--jq", ".content").Output()
	if err != nil {
		return nil, fmt.Errorf("fetch index from %s: %w", r.FullName(), err)
	}
	decoded, err := decodeGitHubContent(out)
	if err != nil {
		return nil, fmt.Errorf("decode index content: %w", err)
	}
	var idx index
	if err := json.Unmarshal(decoded, &idx); err != nil {
		return nil, fmt.Errorf("parse index JSON: %w", err)
	}
	return idx.Skills, nil
}

func CacheIndex(skills []Skill) error {
	cacheFile, err := cacheFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}
	idx := index{Generated: time.Now().UTC().Format(time.RFC3339), Skills: skills}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}
	if err := os.WriteFile(cacheFile, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}
	return nil
}

func LoadCachedIndex() ([]Skill, bool, error) {
	cacheFile, err := cacheFilePath()
	if err != nil {
		return nil, false, err
	}
	info, err := os.Stat(cacheFile)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("stat cache file: %w", err)
	}
	fresh := time.Since(info.ModTime()) < cacheTTL
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false, fmt.Errorf("read cache file: %w", err)
	}
	var idx index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, false, nil
	}
	return idx.Skills, fresh, nil
}

func CacheFilePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache directory: %w", err)
	}
	name := "unconfigured"
	if r, err := ActiveRegistry(); err == nil {
		name = strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(r.FullName())
	}
	return filepath.Join(dir, "hikma", name, "index.json"), nil
}

func cacheFilePath() (string, error) {
	return CacheFilePath()
}

func FetchSkill(name, targetDir string) ([]string, error) {
	r, err := ActiveRegistry()
	if err != nil {
		return nil, err
	}
	return FetchSkillFrom(r, name, targetDir)
}

func FetchSkillFrom(r config.Registry, name, targetDir string) ([]string, error) {
	apiPath := fmt.Sprintf("%s/contents/skills/%s", r.GitHubAPIRepo(), name)
	entries, err := listContents(apiPath)
	if err != nil {
		return nil, fmt.Errorf("fetch skill %q from %s: %w", name, r.FullName(), err)
	}
	return writeSkillFiles(r, entries, fmt.Sprintf("skills/%s", name), targetDir)
}

func DetectLocalChanges(name, localDir string) (bool, error) {
	r, err := ActiveRegistry()
	if err != nil {
		return false, err
	}
	apiPath := fmt.Sprintf("%s/contents/skills/%s", r.GitHubAPIRepo(), name)
	entries, err := listContents(apiPath)
	if err != nil {
		return false, fmt.Errorf("fetch skill %q for change detection: %w", name, err)
	}
	return detectChanges(r, entries, fmt.Sprintf("skills/%s", name), localDir)
}

func listContents(apiPath string) ([]contentEntry, error) {
	out, err := exec.Command("gh", "api", apiPath).Output()
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w", apiPath, err)
	}
	var entries []contentEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("parse contents response: %w", err)
	}
	return entries, nil
}

func detectChanges(r config.Registry, entries []contentEntry, skillRoot, localDir string) (bool, error) {
	registryFiles := make(map[string]bool)
	changed, err := compareRegistryFiles(r, entries, skillRoot, localDir, registryFiles)
	if err != nil || changed {
		return changed, err
	}
	err = filepath.WalkDir(localDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		if !registryFiles[filepath.ToSlash(rel)] {
			changed = true
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("scan local skill directory %s: %w", localDir, err)
	}
	return changed, nil
}

func compareRegistryFiles(r config.Registry, entries []contentEntry, skillRoot, localDir string, registryFiles map[string]bool) (bool, error) {
	for _, e := range entries {
		rel := strings.TrimPrefix(e.Path, skillRoot+"/")
		localPath := filepath.Join(localDir, filepath.FromSlash(rel))
		switch e.Type {
		case "file":
			registryFiles[rel] = true
			registryData, err := fetchFileContent(r, e.Path)
			if err != nil {
				return false, fmt.Errorf("fetch registry file %s: %w", e.Path, err)
			}
			localData, err := os.ReadFile(localPath)
			if os.IsNotExist(err) {
				return true, nil
			}
			if err != nil {
				return false, fmt.Errorf("read local file %s: %w", localPath, err)
			}
			if sha256.Sum256(localData) != sha256.Sum256(registryData) {
				return true, nil
			}
		case "dir":
			subEntries, err := listContents(fmt.Sprintf("%s/contents/%s", r.GitHubAPIRepo(), e.Path))
			if err != nil {
				return false, err
			}
			changed, err := compareRegistryFiles(r, subEntries, skillRoot, localDir, registryFiles)
			if err != nil || changed {
				return changed, err
			}
		}
	}
	return false, nil
}

var fetchFileContent = func(r config.Registry, repoPath string) ([]byte, error) {
	out, err := exec.Command("gh", "api", r.GitHubAPIRepo()+"/contents/"+repoPath, "--jq", ".content").Output()
	if err != nil {
		return nil, fmt.Errorf("gh api %s/contents/%s: %w", r.GitHubAPIRepo(), repoPath, err)
	}
	return decodeGitHubContent(out)
}

func writeSkillFiles(r config.Registry, entries []contentEntry, skillRoot, targetDir string) ([]string, error) {
	var written []string
	for _, e := range entries {
		rel := strings.TrimPrefix(e.Path, skillRoot+"/")
		dst := filepath.Join(targetDir, filepath.FromSlash(rel))
		switch e.Type {
		case "file":
			data, err := fetchFileContent(r, e.Path)
			if err != nil {
				return written, fmt.Errorf("fetch content for %s: %w", e.Path, err)
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return written, fmt.Errorf("create directory for %s: %w", dst, err)
			}
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return written, fmt.Errorf("write %s: %w", dst, err)
			}
			written = append(written, rel)
		case "dir":
			subEntries, err := listContents(fmt.Sprintf("%s/contents/%s", r.GitHubAPIRepo(), e.Path))
			if err != nil {
				return written, err
			}
			subWritten, err := writeSkillFiles(r, subEntries, skillRoot, targetDir)
			if err != nil {
				return written, err
			}
			written = append(written, subWritten...)
		}
	}
	return written, nil
}

func decodeGitHubContent(out []byte) ([]byte, error) {
	encoded := strings.ReplaceAll(strings.TrimSpace(string(out)), "\n", "")
	encoded = strings.Trim(encoded, `"`)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return data, nil
}
