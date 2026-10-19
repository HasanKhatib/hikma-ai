package commands

import (
	"fmt"
	"io"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/hasankhatib/hikma-ai/internal/ui"
)

// registryAccess is what GitHub says about publishing to a registry, found out
// when it is needed and never stored. Write access differs from person to
// person and branch rules change, so a saved copy (above all one in committed
// config) would go stale or say the wrong thing about a teammate.
type registryAccess struct {
	Known         bool   // false when GitHub could not be asked, or did not say
	DefaultBranch string // main, master, or whatever the repository uses
	CanWrite      bool
	RequiresPR    bool // a rule on the default branch requires pull requests
}

// detectRegistryAccess asks GitHub, read-only, how publishing to r would go.
// It needs no sign-in to run, but only a signed-in gh can see write access.
func detectRegistryAccess(r config.Registry) registryAccess {
	canPush, err := ghGet("repos/"+r.FullName(), ".permissions.push")
	if err != nil || (canPush != "true" && canPush != "false") {
		return registryAccess{}
	}
	a := registryAccess{Known: true, CanWrite: canPush == "true"}
	if branch, err := ghGet("repos/"+r.FullName(), ".default_branch"); err == nil {
		a.DefaultBranch = branch
		a.RequiresPR = a.CanWrite && requiresPullRequest(r.FullName(), branch)
	}
	return a
}

// publishLine says how push and import will reach the registry, for the
// confirmation before a write.
func (a registryAccess) publishLine() string {
	switch {
	case !a.Known:
		return "committed to the default branch if you have write access (no review), otherwise a pull request from your fork; add --pr to always open one"
	case !a.CanWrite:
		return "as a pull request from your fork (you have no write access)"
	case a.RequiresPR:
		return "as a pull request (the default branch requires one)"
	}
	return "committed straight to the default branch, with no review (add --pr for a pull request)"
}

// summary is the one line init and doctor print about a registry.
func (a registryAccess) summary() string {
	branch := "the default branch"
	if a.DefaultBranch != "" {
		branch = a.DefaultBranch
	}
	switch {
	case !a.CanWrite:
		return "you cannot write to it, so push and import go through your fork as a pull request"
	case a.RequiresPR:
		return fmt.Sprintf("%s requires pull requests, so push and import open one", branch)
	}
	return fmt.Sprintf("push and import commit straight to %s (you can write, and no rule requires a pull request); add --pr to open one instead", branch)
}

// describeRegistry says how publishing to a registry will go. ok is false, and
// nothing should be printed, when it cannot tell: a registry that is not on
// GitHub, no gh, or not signed in. It is best effort and never an error.
func describeRegistry(registry string) (line string, ok bool) {
	src, err := source.Parse(registry)
	if err != nil || src.Kind != source.KindGitHub {
		return "", false
	}
	a := detectRegistryAccess(config.Registry{Owner: src.Owner, Repo: src.Repo, Raw: src.Raw})
	if !a.Known {
		return "", false
	}
	return a.summary(), true
}

// reportRegistryAccess prints, for init, what publishing to the registry will
// do. It asks nothing and never fails the caller.
func reportRegistryAccess(w io.Writer, registry string) {
	if line, ok := describeRegistry(registry); ok {
		fmt.Fprintf(w, "Registry %s: %s\n", registryName(registry), ui.Dim(w, line))
	}
}

func registryName(registry string) string {
	if src, err := source.Parse(registry); err == nil {
		return src.Display
	}
	return registry
}
