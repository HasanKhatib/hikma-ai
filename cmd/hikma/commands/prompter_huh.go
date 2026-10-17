package commands

import (
	"errors"
	"os"
	"strings"

	"charm.land/huh/v2"
)

// huhPrompter asks through charm.land/huh, which survey's archived upstream no
// longer matches. It is opt-in with HIKMA_PROMPTS=huh until it has been walked
// on Windows PowerShell and the legacy console (E7.3-T9); after that it becomes
// the default and survey goes.
type huhPrompter struct{}

// useHuh reports whether the opt-in huh prompter was asked for.
func useHuh() bool { return strings.EqualFold(os.Getenv("HIKMA_PROMPTS"), "huh") }

// accessible turns the full-screen widgets into plain line prompts, which suits
// a dumb terminal and screen readers.
func accessible() bool { return os.Getenv("TERM") == "dumb" || os.Getenv("ACCESSIBLE") != "" }

func ask(f huh.Field) error {
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
