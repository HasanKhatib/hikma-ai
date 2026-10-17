package commands

import (
	"errors"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/hasankhatib/hikma-ai/internal/ui"
)

// huhPrompter asks through charm.land/huh.
type huhPrompter struct{}

// accessible turns the full-screen widgets into plain line prompts, which suits
// a dumb terminal and screen readers.
func accessible() bool { return os.Getenv("TERM") == "dumb" || os.Getenv("ACCESSIBLE") != "" }

// applyNoColor hands --no-color to huh, which decides its color profile from
// NO_COLOR when it starts. Children (git, gh) read NO_COLOR too, which is what
// someone passing --no-color wants.
func applyNoColor() {
	if ui.NoColor {
		os.Setenv("NO_COLOR", "1")
	}
}

func ask(f huh.Field) error {
	applyNoColor()
	err := huh.NewForm(huh.NewGroup(f)).WithAccessible(accessible()).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return errCancelled
	}
	return err
}

func (huhPrompter) Select(message string, options []string, def int) (int, error) {
	idx := def
	opts := make([]huh.Option[int], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, i)
	}
	err := ask(huh.NewSelect[int]().Title(message).Options(opts...).Height(len(options) + 2).Value(&idx))
	return idx, err
}

func (huhPrompter) MultiSelect(message string, options []string, defaults []int, min int) ([]int, error) {
	var picked []int
	opts := make([]huh.Option[int], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, i)
		for _, d := range defaults {
			if d == i {
				opts[i] = opts[i].Selected(true)
			}
		}
	}
	err := ask(huh.NewMultiSelect[int]().Title(message).Options(opts...).Value(&picked).Validate(func(sel []int) error {
		if len(sel) < min {
			return errors.New("choose at least one")
		}
		return nil
	}))
	return picked, err
}

func (huhPrompter) Confirm(message string, def bool) (bool, error) {
	ok := def
	err := ask(huh.NewConfirm().Title(message).Value(&ok))
	return ok, err
}

func (huhPrompter) Input(message string, validate func(string) error) (string, error) {
	var s string
	err := ask(huh.NewInput().Title(message).Value(&s).Validate(func(v string) error {
		if strings.TrimSpace(v) == "" {
			return errors.New("enter a value")
		}
		if validate != nil {
			return validate(v)
		}
		return nil
	}))
	return s, err
}
