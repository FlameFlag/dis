package main

import (
	"context"
	"dis/cmd"
	"dis/internal/procgroup"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// fang handles the first SIGINT/SIGTERM (cancels ctx).
	// We install a second-signal handler as a backstop: if the user hits
	// Ctrl+C again (or we receive another SIGTERM), forcefully kill all
	// tracked child process groups before exiting.
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch // first signal, handled by fang's NotifyContext
		<-ch // second signal, user is impatient
		procgroup.KillAll()
		os.Exit(1)
	}()

	if err := cmd.Execute(context.Background()); err != nil {
		os.Exit(1)
	}
}
