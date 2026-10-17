//go:build !windows

package e2e

import (
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

// term is a running program on a pseudo-terminal. It answers no terminal
// queries, like a minimal terminal would, so it also covers prompts that wait
// for a reply a terminal never sends.
type term struct {
	t   *testing.T
	cmd *exec.Cmd
	pty interface {
		Write([]byte) (int, error)
		Close() error
	}
	mu   sync.Mutex
	buf  []byte
	done chan struct{}
	exit error
}

func startTerm(t *testing.T, dir string, env []string, args ...string) *term {
	t.Helper()
	cmd := exec.Command(hikmaBinary(t), args...)
	cmd.Dir = dir
	cmd.Env = env
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 110})
	if err != nil {
		t.Fatalf("start on a pty: %v", err)
	}
	tm := &term{t: t, cmd: cmd, pty: f, done: make(chan struct{})}
	go func() {
		chunk := make([]byte, 4096)
		for {
			n, err := f.Read(chunk)
			if n > 0 {
				tm.mu.Lock()
				tm.buf = append(tm.buf, chunk[:n]...)
				tm.mu.Unlock()
			}
			if err != nil {
				break
			}
		}
		tm.exit = cmd.Wait()
		close(tm.done)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = f.Close()
	})
	return tm
}

func (tm *term) write(s string) {
	tm.t.Helper()
	if _, err := tm.pty.Write([]byte(s)); err != nil {
		tm.t.Fatalf("write %q: %v", s, err)
	}
}

func (tm *term) output() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return string(tm.buf)
}

func (tm *term) waitDone(d time.Duration) (exited bool, err error) {
	select {
	case <-tm.done:
		return true, tm.exit
	case <-time.After(d):
		return false, nil
	}
}
