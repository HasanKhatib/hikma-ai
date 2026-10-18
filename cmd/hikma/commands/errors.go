package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// friendlyErrors makes argument and flag mistakes say what is wrong and what
// to type instead. Cobra's own messages ("accepts 1 arg(s), received 0") name
// the rule, not the fix. It walks every command so new commands get it too.
func friendlyErrors(root *cobra.Command) {
	root.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		msg := err.Error()
		if name, ok := strings.CutPrefix(msg, "unknown flag: --"); ok {
			var flags []string
			c.Flags().VisitAll(func(f *pflag.Flag) { flags = append(flags, f.Name) })
			if s := closest(name, flags); s != "" {
				msg += fmt.Sprintf(" (did you mean --%s?)", s)
			}
		}
		return usageError(fmt.Errorf("%s\nUsage: %s\nRun '%s --help' to see the flags", msg, c.UseLine(), c.CommandPath()))
	})
	var wrap func(c *cobra.Command)
	wrap = func(c *cobra.Command) {
		switch {
		case c.HasSubCommands() && c.Args == nil:
			c.Args = unknownSubcommand
			if !c.Runnable() {
				c.RunE = func(c *cobra.Command, _ []string) error { return c.Help() }
			}
		case c.Args != nil && !c.HasSubCommands() && c.Annotations[ownArgErrors] != "true":
			c.Args = friendlyArgs(c.Args)
		case c.Args != nil && !c.HasSubCommands():
			inner := c.Args
			c.Args = func(c *cobra.Command, args []string) error { return usageError(inner(c, args)) }
		}
		for _, sub := range c.Commands() {
			wrap(sub)
		}
	}
	wrap(root)
}

// ownArgErrors marks a command whose Args validator already words its errors.
const ownArgErrors = "ownArgErrors"

func unknownSubcommand(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	var names []string
	for _, sub := range c.Commands() {
		if !sub.Hidden {
			names = append(names, sub.Name())
		}
	}
	msg := fmt.Sprintf("unknown command %q for '%s'", args[0], c.CommandPath())
	if s := closest(args[0], names); s != "" {
		msg += fmt.Sprintf(". Did you mean '%s %s'?", c.CommandPath(), s)
	}
	return usageError(fmt.Errorf("%s\nRun '%s --help' to list the commands", msg, c.CommandPath()))
}

func friendlyArgs(inner cobra.PositionalArgs) cobra.PositionalArgs {
	return func(c *cobra.Command, args []string) error {
		err := inner(c, args)
		if err == nil {
			return nil
		}
		var what string
		switch {
		case len(args) == 0:
			what = "missing argument"
		case inner(c, nil) == nil:
			what = fmt.Sprintf("unexpected argument %q", strings.Join(args, " "))
			if len(args) > 1 {
				what = fmt.Sprintf("unexpected arguments %q", strings.Join(args, " "))
			}
		default:
			what = fmt.Sprintf("wrong number of arguments (got %d)", len(args))
		}
		return usageError(fmt.Errorf("%s for '%s'\nUsage: %s\nRun '%s --help' for examples", what, c.CommandPath(), c.UseLine(), c.CommandPath()))
	}
}

// closest returns the candidate within a small edit distance of word, or "".
func closest(word string, candidates []string) string {
	best, bestD := "", 3
	for _, c := range candidates {
		if d := editDistance(strings.ToLower(word), c); d < bestD {
			best, bestD = c, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
