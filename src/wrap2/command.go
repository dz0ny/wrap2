package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

// Command holds data about process to be executed
type Command struct {
	Command  string   `toml:"cmd"`
	Template Template `toml:"config, omitempty"`
}

// Run executes process and redirects pipes
func (c *Command) Run(ctx context.Context, cancel context.CancelFunc) {
	defer wg.Done()
	args := strings.Split(c.Command, " ")
	process := exec.Command(args[0], args[1:]...)
	go io.Copy(process.Stdout, os.Stdout)
	go io.Copy(process.Stderr, os.Stderr)

	// start the process
	err := process.Start()
	if err != nil {
		log.Error(
			"Failed starting command",
			zap.String("cmd", c.Command),
			zap.Error(err),
		)
	}

	// Setup signaling
	catch := make(chan os.Signal, 1)
	signal.Notify(catch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case sig := <-catch:
			log.Info(
				"Command timed out",
				zap.String("cmd", c.Command),
			)
			signalProcessWithTimeout(process, sig)
			cancel()
		case <-ctx.Done():
			// exit when context is done
		}
	}()

	err = process.Wait()
	cancel()

	if err != nil {
		log.Info(
			"Command terminated",
			zap.String("cmd", c.Command),
			zap.Error(err),
		)
		// OPTIMIZE: This could be cleaner
		os.Exit(err.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus())
	}

	log.Info(
		"Command ended",
		zap.String("cmd", c.Command),
	)

}
