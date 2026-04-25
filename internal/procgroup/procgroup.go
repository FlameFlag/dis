//go:build unix

package procgroup

import (
	"os/exec"
	"sync"
	"syscall"
	"time"
)

var (
	mu     sync.Mutex
	active = map[*exec.Cmd]struct{}{}
)

// Setup configures cmd to run in its own process group and, on context
// cancellation, sends SIGTERM to the entire group (including grandchildren
// like ffmpeg). Go's WaitDelay escalates to SIGKILL if the group doesn't
// exit within gracePeriod.
//
// The command is tracked so KillAll can clean up on abrupt shutdown.
func Setup(cmd *exec.Cmd, gracePeriod time.Duration) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	cmd.WaitDelay = gracePeriod
}

// Track registers a started command for cleanup. Call after cmd.Start().
func Track(cmd *exec.Cmd) {
	mu.Lock()
	active[cmd] = struct{}{}
	mu.Unlock()
}

// Untrack removes a command from cleanup tracking. Call after cmd.Wait().
func Untrack(cmd *exec.Cmd) {
	mu.Lock()
	delete(active, cmd)
	mu.Unlock()
}

// KillAll sends SIGKILL to every tracked process group. Safe to call from
// a signal handler, it does not block on process exit.
func KillAll() {
	mu.Lock()
	defer mu.Unlock()
	for cmd := range active {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}
}
