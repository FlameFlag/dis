package procgroup

import (
	"os/exec"
	"sync"
	"time"
)

var (
	mu     sync.Mutex
	active = map[*exec.Cmd]struct{}{}
)

func Setup(cmd *exec.Cmd, gracePeriod time.Duration) {
	cmd.WaitDelay = gracePeriod
}

func Track(cmd *exec.Cmd) {
	mu.Lock()
	active[cmd] = struct{}{}
	mu.Unlock()
}

func Untrack(cmd *exec.Cmd) {
	mu.Lock()
	delete(active, cmd)
	mu.Unlock()
}

func KillAll() {
	mu.Lock()
	defer mu.Unlock()
	for cmd := range active {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
}
