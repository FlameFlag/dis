package procgroup

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Run executes cmd with the standard process-group lifecycle:
// Setup(gracePeriod) → Start → Track → (onStart) → Wait → Untrack.
//
// If onStart is non-nil, it runs between Start and Wait — use it to consume
// pipes (e.g. StderrPipe scanning) that were wired up before the call. Any
// error it returns is preferred over Wait's error.
//
// Pipes (StderrPipe/StdoutPipe) must be established on cmd before calling Run,
// since cmd.Start happens inside.
func Run(_ context.Context, cmd *exec.Cmd, gracePeriod time.Duration, onStart func() error) error {
	Setup(cmd, gracePeriod)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}
	Track(cmd)
	defer Untrack(cmd)

	var readErr error
	if onStart != nil {
		readErr = onStart()
	}

	waitErr := cmd.Wait()
	if readErr != nil {
		return readErr
	}
	return waitErr
}
