package main

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

// Command holds data about process to be executed
type Command struct {
	Command  string   `toml:"cmd"`
	Template Template `toml:"config, omitempty"`
	RunAs    string   `toml:"user, omitempty"`
	Enabled  Enabler  `toml:"enabled, omitempty"`
	uid      int
	gid      int
	user     string
}

type logger struct {
	Kind    string
	Command string
}

func (l logger) Write(data []byte) (int, error) {

	// if we got json data just write it out
	if string(data[0]) == "{" {
		return os.Stdout.Write(data)
	}

	// wrap in json otherwise
	log.Warn(
		string(data),
		zap.String("kind", l.Kind),
		zap.String("cmd", l.Command),
	)
	return len(data), nil
}

// RunBlocking runs command in blocking mode
func (c *Command) RunBlocking(fatal bool, ctx context.Context) {

	args := strings.Split(c.Command, " ")
	process := exec.Command(args[0], args[1:]...)
	process.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if c.RunAs != "" {
		currentUser, err := user.Lookup(c.RunAs)
		if err != nil {
			log.Fatal(
				"Failed getting user",
				zap.String("run_as", c.RunAs),
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		}
		c.user = currentUser.Username
		c.uid, _ = strconv.Atoi(currentUser.Uid)
		c.gid, _ = strconv.Atoi(currentUser.Gid)
		process.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(c.uid), Gid: uint32(c.gid)}
	}

	process.Stdout = logger{"stdout", c.Command}
	process.Stderr = logger{"stderr", c.Command}
	process.Stdin = nil
	log.Info("Starting blocking", zap.Strings("args", args), zap.String("user", c.user))

	err := process.Start()
	if err != nil {
		log.Fatal(
			"Failed starting command",
			zap.String("cmd", c.Command),
			zap.Error(err),
		)
	}
	innerCtx, cancelWatcher := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-innerCtx.Done():
				log.Debug("Done", zap.Int("pid", process.Process.Pid))
				return
			case <-ctx.Done():
				log.Debug("Terminating blocking", zap.Int("pid", process.Process.Pid))
				err := process.Process.Kill()
				if err != nil {
					log.Debug("Terminating blocking failed", zap.Int("pid", process.Process.Pid), zap.Error(err))
				}
				return
			}
		}
	}()
	err = process.Wait()
	cancelWatcher()

	// no *child processes is normal error* when we are pid 1
	if err != nil && !strings.Contains(err.Error(), "no child processes") {
		if fatal {
			log.Fatal(
				"Process terminated",
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		} else {
			log.Debug(
				"Process terminated",
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		}
	} else {
		log.Debug(
			"Process ended",
			zap.String("cmd", c.Command),
		)
	}
}

// RunBlockingNonFatal runs command in blocking mode
func (c *Command) RunBlockingNonFatal(ctx context.Context) {
	c.RunBlocking(false, ctx)
}

// Run executes process and redirects pipes
func (c *Command) Run(ctx context.Context) {
	// Register chan to receive system signals
	wg.Add(1)
	args := strings.Split(c.Command, " ")
	process := exec.Command(args[0], args[1:]...)
	process.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if c.RunAs != "" {
		currentUser, err := user.Lookup(c.RunAs)
		if err != nil {
			log.Fatal(
				"Failed getting user",
				zap.String("run_as", c.RunAs),
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		}
		c.user = currentUser.Username
		c.uid, _ = strconv.Atoi(currentUser.Uid)
		c.gid, _ = strconv.Atoi(currentUser.Gid)
		process.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(c.uid), Gid: uint32(c.gid)}
	}
	process.Stdout = logger{"stdout", c.Command}
	process.Stderr = logger{"stderr", c.Command}
	process.Stdin = nil

	go func() {
		<-ctx.Done()
		// Done returns a channel that's closed when work done on behalf of this context is canceled
		log.Debug("Terminating forked", zap.Int("pid", process.Process.Pid), zap.String("cmd", c.Command))
		err := process.Process.Kill()
		if err != nil {
			log.Debug("Terminating forked failed", zap.Int("pid", process.Process.Pid), zap.Error(err))
		}
	}()

	go func() {
		log.Info("Starting forked", zap.Strings("args", args), zap.String("user", c.user))

		// start the process
		err := process.Start()
		if err != nil {
			log.Error(
				"Failed starting command",
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		}

		err = process.Wait()
		if err != nil {
			log.Debug(
				"Process terminated",
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		}
		wg.Done()
	}()
}
