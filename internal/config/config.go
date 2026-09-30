package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Agent represents the user-facing AI agent selection.
type Agent string

const (
	AgentCopilot  Agent = "copilot"
	AgentCodex    Agent = "codex"
	AgentOpenCode Agent = "opencode"
	AgentClaude   Agent = "claude"
)

// AgentDescriptions maps each valid agent to a short, user-facing description.
var AgentDescriptions = map[Agent]string{
	AgentCopilot:  "GitHub Copilot using AGENTS.md and .agents/skills",
	AgentCodex:    "Codex CLI using AGENTS.md and .agents/skills",
	AgentOpenCode: "OpenCode using AGENTS.md and .agents/skills",
	AgentClaude:   "Claude Code using CLAUDE.md and .claude/skills",
}

// ValidAgents lists all recognised user-facing agent values.
var ValidAgents = []Agent{AgentCopilot, AgentCodex, AgentOpenCode, AgentClaude}

// Profile represents an internal AI tool layout value.
type Profile string

const (
	ProfileDefault Profile = "default"
	ProfileClaude  Profile = "claude"
)

// ValidProfiles lists all recognised profile values.
var ValidProfiles = []Profile{ProfileDefault, ProfileClaude}

// Config holds user-level hikma configuration.
type Config struct {
	Agent    Agent  `json:"agent,omitempty"`
	Registry string `json:"registry,omitempty"`
}

// Selection is the resolved agent identity and layout profile.
type Selection struct {
	Agent   Agent
	Profile Profile
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
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("cannot read config file: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("cannot parse config file: %w", err)
	}
	return c, nil
}

// Save writes c to os.UserConfigDir()/hikma/config.json.
func Save(c Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("cannot write config file: %w", err)
	}
	return nil
}

func ConfigForAgent(a Agent) (Config, error) {
	if !ValidAgent(a) {
		return Config{}, fmt.Errorf("unsupported agent %q - supported values: copilot, codex, opencode, claude", a)
	}
	c, err := Load()
	if err != nil {
		return Config{}, err
	}
	c.Agent = a
	return c, nil
}

func ConfigForRegistry(value string) (Config, error) {
	if _, err := ParseRegistry(value); err != nil {
		return Config{}, err
	}
	c, err := Load()
	if err != nil {
		return Config{}, err
	}
	c.Registry = value
	return c, nil
}

// ResolveSelection returns the effective agent selection using flag, env, config, then copilot.
func ResolveSelection(flagAgent string) (Selection, error) {
	if flagAgent != "" {
		return selectionFromAgent(flagAgent)
	}
	if env := os.Getenv("HIKMA_AGENT"); env != "" {
		return selectionFromAgent(env)
	}
	c, err := Load()
	if err != nil {
		return Selection{}, err
	}
	if c.Agent != "" {
		if !ValidAgent(c.Agent) {
			return Selection{}, fmt.Errorf("unsupported agent %q in config file - supported values: copilot, codex, opencode, claude\nrun 'hikma config agent' to reconfigure", c.Agent)
		}
		return Selection{Agent: c.Agent, Profile: ProfileForAgent(c.Agent)}, nil
	}
	return Selection{Agent: AgentCopilot, Profile: ProfileDefault}, nil
}

func ResolveRegistry(flagRegistry string) (Registry, error) {
	if flagRegistry != "" {
		return ParseRegistry(flagRegistry)
	}
	if env := os.Getenv("HIKMA_REGISTRY"); env != "" {
		return ParseRegistry(env)
	}
	c, err := Load()
	if err != nil {
		return Registry{}, err
	}
	if c.Registry != "" {
		return ParseRegistry(c.Registry)
	}
	return Registry{}, fmt.Errorf("no registry configured - run 'hikma config registry <owner/repo>' or pass --registry")
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
		return Selection{}, fmt.Errorf("unsupported agent %q - supported values: copilot, codex, opencode, claude", val)
	}
	return Selection{Agent: a, Profile: ProfileForAgent(a)}, nil
}

func ValidAgent(a Agent) bool {
	for _, v := range ValidAgents {
		if a == v {
			return true
		}
	}
	return false
}

func ProfileForAgent(a Agent) Profile {
	if a == AgentClaude {
		return ProfileClaude
	}
	return ProfileDefault
}

func Valid(p Profile) bool {
	for _, v := range ValidProfiles {
		if p == v {
			return true
		}
	}
	return false
}

func SkillDir(name string, p Profile) string {
	return filepath.Join(SkillBasePath(p), name)
}

func SkillBasePath(p Profile) string {
	switch p {
	case ProfileClaude:
		return ".claude/skills"
	default:
		return ".agents/skills"
	}
}

func AgentSkillDir(name string, a Agent) string {
	return SkillDir(name, ProfileForAgent(a))
}
