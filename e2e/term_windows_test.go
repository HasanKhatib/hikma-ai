//go:build windows

package e2e

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/x/conpty"
)

// term is a running program on a Windows pseudo-console (ConPTY), the layer
// Windows Terminal and the legacy console host both sit on. It answers no
// terminal queries of its own.
type term struct {
	t    *testing.T
	pty  *conpty.ConPty
	proc *os.Process
	mu   sync.Mutex
	buf  []byte
	done chan struct{}
	exit error
}

func startTerm(t *testing.T, dir string, env []string, args ...string) *term {
	t.Helper()
	bin := hikmaBinary(t)
	c, err := conpty.New(110, 40, 0)
	if err != nil {
		t.Fatalf("create a pseudo-console: %v", err)
	}
	pid, _, err := c.Spawn(bin, append([]string{bin}, args...), &syscall.ProcAttr{Dir: dir, Env: env})
	if err != nil {
		_ = c.Close()
		t.Fatalf("start on a pseudo-console: %v", err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = c.Close()
		t.Fatalf("find process %d: %v", pid, err)
	}
	tm := &term{t: t, pty: c, proc: proc, done: make(chan struct{})}
	go func() {
		chunk := make([]byte, 4096)
		for {
			n, err := c.Read(chunk)
			if n > 0 {
				tm.mu.Lock()
				tm.buf = append(tm.buf, chunk[:n]...)
				tm.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		st, err := proc.Wait()
		switch {
		case err != nil:
			tm.exit = err
		case !st.Success():
			tm.exit = fmt.Errorf("exit status %d", st.ExitCode())
		}
		// Let the last output drain before callers read it.
		time.Sleep(300 * time.Millisecond)
		close(tm.done)
	}()
	t.Cleanup(func() {
		_ = proc.Kill()
		_ = c.Close()
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
