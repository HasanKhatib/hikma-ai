package config

import (
	"fmt"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"os"
	"path/filepath"
	"strings"
)

// Config holds user-level hikma configuration.
type Config struct {
	Agent    Agent  `json:"agent,omitempty"`
	Registry string `json:"registry,omitempty"`
	Naming   string `json:"naming,omitempty"`
	// Agents lists every agent a project installs skills for.
	Agents []string `json:"agents,omitempty"`
}

// Registry is the normalized GitHub registry identity.
type Registry struct {
	Owner string
	Repo  string
	Raw   string
}

func (r Registry) FullName() string {
	return r.Owner + "/" + r.Repo
}

func (r Registry) GitHubAPIRepo() string {
	return "repos/" + r.FullName()
}

func (r Registry) CloneURL() string {
	return "https://github.com/" + r.FullName() + ".git"
}

// ConfigFilePath returns the path to the user config file.
func ConfigFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine user config directory: %w", err)
	}
	return filepath.Join(dir, "hikma", "config.json"), nil
}

func configFilePath() (string, error) {
	return ConfigFilePath()
}

// Load reads the user config from os.UserConfigDir()/hikma/config.json.
func Load() (Config, error) {
	path, err := configFilePath()
	if err != nil {
		return Config{}, err
	}
	return loadFile(path)
}

// Save writes c to os.UserConfigDir()/hikma/config.json.
func Save(c Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	return saveFile(path, c)
}

func ConfigForAgent(a Agent) (Config, error) {
	if !ValidAgent(a) {
		return Config{}, fmt.Errorf("unsupported agent %q - supported values: %s", a, agentList())
	}
	c, err := Load()
	if err != nil {
		return Config{}, err
	}
	c.Agent = a
	return c, nil
}

func ConfigForRegistry(value string) (Config, error) {
	if _, err := source.Parse(value); err != nil {
		return Config{}, err
	}
	c, err := Load()
	if err != nil {
		return Config{}, err
	}
	c.Registry = value
	return c, nil
}

// ResolveSelection returns the effective agent selection using flag, env, project config, user config, then copilot.
func ResolveSelection(flagAgent string) (Selection, error) {
	if flagAgent != "" {
		return selectionFromAgent(flagAgent)
	}
	e, err := Lookup(KeyAgent)
	if err != nil {
		return Selection{}, err
	}
	if e.Source == SourceEnv {
		return selectionFromAgent(e.Value)
	}
	a := Agent(e.Value)
	if !ValidAgent(a) {
		return Selection{}, fmt.Errorf("unsupported agent %q in %s config - supported values: %s\nrun 'hikma config set agent <value>' to reconfigure", a, e.Source, agentList())
	}
	return SelectionFor(a), nil
}

// ResolveRegistry returns the effective registry using flag, env, project config, then user config.
func ResolveRegistry(flagRegistry string) (Registry, error) {
	if flagRegistry != "" {
		return ParseRegistry(flagRegistry)
	}
	e, err := Lookup(KeyRegistry)
	if err != nil {
		return Registry{}, err
	}
	if e.Value == "" {
		return Registry{}, fmt.Errorf("no registry configured - run 'hikma config set registry <owner/repo>' or pass --registry")
	}
	return ParseRegistry(e.Value)
}

func ParseRegistry(value string) (Registry, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return Registry{}, fmt.Errorf("registry cannot be empty")
	}
	trimmed := strings.TrimSuffix(raw, ".git")
	trimmed = strings.TrimPrefix(trimmed, "https://github.com/")
	trimmed = strings.TrimPrefix(trimmed, "http://github.com/")
	trimmed = strings.TrimPrefix(trimmed, "git@github.com:")
	trimmed = strings.Trim(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Registry{}, fmt.Errorf("registry must be a GitHub owner/repo, HTTPS URL, or SSH URL")
	}
	return Registry{Owner: parts[0], Repo: parts[1], Raw: raw}, nil
}

func selectionFromAgent(val string) (Selection, error) {
	a := Agent(val)
	if !ValidAgent(a) {
		return Selection{}, fmt.Errorf("unsupported agent %q - supported values: %s", val, agentList())
	}
	return SelectionFor(a), nil
}
