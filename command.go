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
	log.Debug(
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
	log.Debug("Starting", zap.Strings("args", args), zap.String("user", c.user))

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
				log.Debug("Terminating", zap.Int("pid", process.Process.Pid))
				process.Process.Kill()
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
	go func(command, runAs string, ctx context.Context) {

		defer wg.Done()
		args := strings.Split(command, " ")
		process := exec.Command(args[0], args[1:]...)
		process.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if runAs != "" {
			currentUser, err := user.Lookup(runAs)
			if err != nil {
				log.Fatal(
					"Failed getting user",
					zap.String("run_as", runAs),
					zap.String("cmd", command),
					zap.Error(err),
				)
			}
			c.user = currentUser.Username
			c.uid, _ = strconv.Atoi(currentUser.Uid)
			c.gid, _ = strconv.Atoi(currentUser.Gid)
			process.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(c.uid), Gid: uint32(c.gid)}
		}

		process.Stdout = logger{"stdout", command}
		process.Stderr = logger{"stderr", command}
		process.Stdin = nil
		log.Debug("Starting", zap.Strings("args", args), zap.String("user", c.user))

		// start the process
		err := process.Start()
		if err != nil {
			log.Error(
				"Failed starting command",
				zap.String("cmd", command),
				zap.Error(err),
			)
		}

		go func() {
			for {
				select {
				case <-ctx.Done(): // Done returns a channel that's closed when work done on behalf of this context is canceled
					log.Debug("Terminating", zap.Int("pid", process.Process.Pid), zap.String("cmd", c.Command))
					process.Process.Kill()
					return
				}
			}
		}()

		err = process.Wait()
		if err != nil {
			log.Debug(
				"Process terminated",
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		}

	}(c.Command, c.RunAs, ctx)
}
