package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/charmbracelet/x/term"
	"github.com/hasankhatib/hikma-ai/internal/ui"
)

// filterAbove is how long a list gets before it can be filtered by typing.
const filterAbove = 8

// huhPrompter asks through charm.land/huh.
type huhPrompter struct{}

// accessible turns the full-screen widgets into plain line prompts, which suits
// a dumb terminal and screen readers.
func accessible() bool { return os.Getenv("TERM") == "dumb" || os.Getenv("ACCESSIBLE") != "" }

// applyNoColor hands --no-color to huh, which decides its color profile from
// NO_COLOR when it starts. Children (git, gh) read NO_COLOR too, which is what
// someone passing --no-color wants.
func applyNoColor() {
	if ui.NoColor && os.Getenv("NO_COLOR") == "" {
		os.Setenv("NO_COLOR", "1")
	}
}

// listHeight is the rows a list of n options takes, with extra rows for the
// title and hint, held to what the terminal can show. When the size is unknown
// the full list height is used, which is what a short list wants anyway.
func listHeight(n, extra int) int {
	_, rows, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		rows = 0
	}
	return fitHeight(n+extra, rows)
}

// fitHeight holds h rows to a terminal of the given rows, leaving room for the
// banner and the lines above the menu, but never fewer than minRows so the list
// can still be used on a very short terminal. rows <= 0 means the size is unknown.
func fitHeight(h, rows int) int {
	const minRows = 4
	if rows <= 0 {
		return h
	}
	limit := max(rows-8, minRows)
	if h > limit {
		return limit
	}
	return h
}

// ask runs one field. Esc backs out like Ctrl-C when esc is true; a list that can
// be filtered keeps Esc for clearing the filter, which huh would otherwise
// pre-empt, since it checks the form's quit key first.
func ask(f huh.Field, esc bool) error {
	applyNoColor()
	km := huh.NewDefaultKeyMap()
	if esc {
		km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
	}
	err := huh.NewForm(huh.NewGroup(f)).WithKeyMap(km).WithAccessible(accessible()).Run()
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
	hint := "enter to choose · esc back"
	filter := len(options) > filterAbove
	if filter {
		hint = "enter to choose · / to filter"
	}
	if accessible() {
		hint = "enter the number of your choice"
	}
	// huh's Select always filters on "/"; the hint only names it on long lists.
	err := ask(huh.NewSelect[int]().Title(message).Description(hint).Options(opts...).
		Height(listHeight(len(options), 3)).Value(&idx), !filter)
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
	hint := "space to tick · ctrl+a for all · enter to confirm · esc back"
	filter := len(options) > filterAbove
	if filter {
		hint = "space to tick · ctrl+a for all · enter to confirm · / to filter"
	}
	if accessible() {
		hint = "enter numbers separated by commas"
	}
	err := ask(huh.NewMultiSelect[int]().Title(message).Description(hint).Options(opts...).
		Filterable(filter).Height(listHeight(len(options), 3)).Value(&picked).Validate(func(sel []int) error {
		if len(sel) < min {
			return fmt.Errorf("choose at least %d", min)
		}
		return nil
	}), !filter)
	return picked, err
}

func (huhPrompter) Confirm(message string, def bool) (bool, error) {
	ok := def
	err := ask(huh.NewConfirm().Title(message).Value(&ok), true)
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
	}), true)
	return s, err
}

func (huhPrompter) Spinner(message string, fn func(ctx context.Context) error) error {
	if accessible() {
		// The widget is animated; a screen reader and a dumb terminal get a line.
		fmt.Println(message)
		return fn(context.Background())
	}
	applyNoColor()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := spinner.New().Title(message).Context(ctx).ActionWithErr(fn).Run()
	if errors.Is(err, tea.ErrInterrupted) || errors.Is(err, context.Canceled) {
		return errCancelled
	}
	return err
}
