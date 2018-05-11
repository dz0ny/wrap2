package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Command holds data about process to be executed
type Command struct {
	Command  string   `toml:"cmd"`
	Template Template `toml:"config, omitempty"`
}

// Run executes process and redirects pipes
func (c *Command) Run(ctx context.Context, cancel context.CancelFunc) {
	go func(command string) {
		defer wg.Done()
		args := strings.Split(command, " ")
		process := exec.Command(args[0], args[1:]...)
		process.Stdout = os.Stdout
		process.Stderr = os.Stderr
		process.Stdin = os.Stdin
		log.Info("Starting", zap.Strings("args", args))
		// start the process
		err := process.Start()
		if err != nil {
			log.Error(
				"Failed starting command",
				zap.String("cmd", command),
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
					"Terminating",
					zap.String("cmd", command),
					zap.String("signal", sig.String()),
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
				zap.String("cmd", command),
				zap.Error(err),
			)
			// OPTIMIZE: This could be cleaner
			os.Exit(err.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus())
		}
	}(c.Command)
}

func signalProcessWithTimeout(process *exec.Cmd, sig os.Signal) {
	done := make(chan bool)
	go func() {
		process.Process.Signal(syscall.SIGINT)
		process.Process.Signal(syscall.SIGTERM)
		process.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		log.Info(
			"Command termianted due to timeout",
			zap.String("cmd", process.Path),
		)
		process.Process.Kill()
	}
}
