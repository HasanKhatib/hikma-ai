package commands

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/spf13/cobra"
)

// skillResult is what happened to one skill in a multi-skill install.
type skillResult struct {
	Name string
	Err  error
}

// installFailed is returned when some, but not all, of the skills named with
// --skill could not be installed. The menu reads Failed to offer a retry.
type installFailed struct {
	Failed []string
	Total  int
}

// Error is short on purpose: the summary above it already names each skill.
func (e *installFailed) Error() string {
	return fmt.Sprintf("%d of %d skills were not installed", len(e.Failed), e.Total)
}

// renderResult is the summary of a multi-skill install: one line per skill with a
// word for its state, so nothing depends on color.
func renderResult(res []skillResult) string {
	ok := 0
	for _, r := range res {
		if r.Err == nil {
			ok++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Installed %d of %d\n", ok, len(res))
	width := 0
	for _, r := range res {
		width = max(width, ui.Width(r.Name))
	}
	for _, r := range res {
		if r.Err == nil {
			fmt.Fprintf(&b, "  %-6s  %s\n", "ok", r.Name)
			continue
		}
		// An error's later lines often say what to do ("use --force to reinstall").
		var parts []string
		for _, l := range strings.Split(r.Err.Error(), "\n") {
			if l = strings.TrimSpace(l); l != "" {
				parts = append(parts, l)
			}
		}
		reason := strings.Join(parts, "; ")
		fmt.Fprintf(&b, "  %-6s  %s  %s\n", "failed", ui.Pad(r.Name, width), reason)
	}
	return b.String()
}

// installMany installs every skill named with --skill from the one checkout that is
// already open. Names are checked first, so a typo writes nothing. The write
// guard and the confirmation cover all of them at once, the same as for one skill.
func installMany(cmd *cobra.Command, sc scope, src source.Source, co source.Checkout, skills []source.Skill, names []string, targets []config.Selection, force, yes bool, ref string) error {
	out := cmd.OutOrStdout()

	seen := map[string]bool{}
	var picked []source.Skill
	var missing []string
	for _, n := range names {
		n = strings.TrimPrefix(strings.TrimSpace(n), "@")
		if n == "" || seen[strings.ToLower(n)] {
			continue
		}
		seen[strings.ToLower(n)] = true
		skill, ok := source.Find(skills, n)
		if !ok {
			missing = append(missing, n)
			continue
		}
		picked = append(picked, skill)
	}
	if len(missing) > 0 {
		printSkillTable(cmd.ErrOrStderr(), skills)
		sort.Strings(missing)
		return fmt.Errorf("not found in %s: %s\nnothing was installed", src.Display, strings.Join(missing, ", "))
	}
	if len(picked) == 0 {
		return usageError(fmt.Errorf("--skill needs a skill name"))
	}

	var keys, installNames []string
	for _, s := range picked {
		installNames = append(installNames, s.Name)
		for _, t := range targets {
			keys = append(keys, sc.key(t, s.Name))
		}
	}
	if err := sc.guardWrite(keys...); err != nil {
		return err
	}
	for _, w := range sc.shadowWarnings(installNames) {
		ui.PrintWarning(out, w)
	}
	if err := sc.confirmWrite(cmd, yes, keys); err != nil {
		return err
	}

	ctx := cmd.Context()
	res := make([]skillResult, 0, len(picked))
	for _, s := range picked {
		// A cancel stops between skills: the ones already copied stay, the rest
		// are reported as not started.
		if err := ctx.Err(); err != nil {
			res = append(res, skillResult{Name: s.Name, Err: errors.New("not started: cancelled")})
			continue
		}
		res = append(res, skillResult{Name: s.Name, Err: installSkill(sc, out, src, co, s, targets, force, ref)})
	}
	if len(res) == 1 {
		return res[0].Err
	}
	fmt.Fprintln(out)
	fmt.Fprint(out, renderResult(res))
	if err := ctx.Err(); err != nil {
		return err
	}
	var failed []string
	for _, r := range res {
		if r.Err != nil {
			failed = append(failed, r.Name)
		}
	}
	if len(failed) > 0 {
		return &installFailed{Failed: failed, Total: len(res)}
	}
	return nil
}
