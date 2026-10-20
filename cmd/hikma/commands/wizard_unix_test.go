//go:build !windows

package commands

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"
)

// Work with no child process (copying files, say) has nothing the terminal can
// stop for it, so Ctrl-C has to reach it as a cancelled context.
func TestWithInterruptCancelsWorkThatHasNoChildProcess(t *testing.T) {
	err := withInterrupt(func(ctx context.Context) error {
		if kerr := syscall.Kill(syscall.Getpid(), syscall.SIGINT); kerr != nil {
			t.Fatal(kerr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return errors.New("Ctrl-C did not reach the work")
		}
	})
	if !errors.Is(err, errCancelled) {
		t.Fatalf("got %v, want errCancelled so the menu comes back", err)
	}
}

func TestWithInterruptKeepsARealError(t *testing.T) {
	boom := errors.New("boom")
	if err := withInterrupt(func(context.Context) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("got %v, want the command's own error", err)
	}
}
