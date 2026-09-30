package preflight

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Failure struct {
	Name string
	Err  string
	Fix  []string
}

type Result struct {
	Passed   []string
	Failures []Failure
}

func (r Result) OK() bool { return len(r.Failures) == 0 }

func Check() Result {
	var result Result
	checks := []struct {
		name string
		run  func() (string, *Failure)
	}{
		{name: "gh installed", run: checkGHInstalled},
		{name: "gh authenticated", run: checkGHAuthenticated},
		{name: "gh version", run: checkGHVersion},
		{name: "git installed", run: checkGitInstalled},
	}
	for _, c := range checks {
		detail, failure := c.run()
		if failure != nil {
			result.Failures = append(result.Failures, *failure)
			continue
		}
		label := c.name
		if detail != "" {
			label = fmt.Sprintf("%s (%s)", c.name, detail)
		}
		result.Passed = append(result.Passed, label)
	}
	return result
}

func checkGHInstalled() (string, *Failure) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", &Failure{Name: "gh installed", Err: "gh CLI not found on PATH", Fix: []string{"brew install gh"}}
	}
	return "", nil
}

func checkGHAuthenticated() (string, *Failure) {
	out, err := exec.Command("gh", "auth", "status", "--hostname", "github.com").CombinedOutput()
	if err != nil {
		return "", &Failure{Name: "gh authenticated", Err: "not authenticated", Fix: []string{"gh auth login"}}
	}
	return parseGHUser(string(out)), nil
}

func checkGHVersion() (string, *Failure) {
	out, err := exec.Command("gh", "--version").Output()
	if err != nil {
		return "", &Failure{Name: "gh version", Err: "could not determine gh version", Fix: []string{"brew upgrade gh"}}
	}
	version, major := parseGHVersion(string(out))
	if major < 2 {
		return "", &Failure{Name: "gh version", Err: fmt.Sprintf("gh version %s is below minimum required (2.x)", version), Fix: []string{"brew upgrade gh"}}
	}
	return version, nil
}

func checkGitInstalled() (string, *Failure) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", &Failure{Name: "git installed", Err: "git not found on PATH", Fix: []string{"brew install git"}}
	}
	return "", nil
}

func parseGHUser(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Logged in to") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "account" && i+1 < len(parts) {
					return strings.TrimSuffix(parts[i+1], "(keyring)")
				}
			}
		}
	}
	return ""
}

func parseGHVersion(output string) (string, int) {
	line := strings.SplitN(output, "\n", 2)[0]
	parts := strings.Fields(line)
	for i, p := range parts {
		if p == "version" && i+1 < len(parts) {
			v := parts[i+1]
			major := 0
			if dot := strings.Index(v, "."); dot > 0 {
				major, _ = strconv.Atoi(v[:dot])
			}
			return v, major
		}
	}
	return "", 0
}
