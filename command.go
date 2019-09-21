package main

import (
	"os"
	"os/exec"
	"os/signal"
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
	log.Info(
		string(data),
		zap.String("kind", l.Kind),
		zap.String("cmd", l.Command),
	)
	return len(data), nil
}

// RunBlocking runs command in blocking mode
func (c *Command) RunBlocking(fatal bool) {

	// Register chan to receive system signals
	commandSig := make(chan os.Signal, 1)
	defer close(commandSig)
	signal.Notify(commandSig)
	defer signal.Reset()

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
	log.Info("Starting", zap.Strings("args", args), zap.String("user", c.user))

	// Goroutine for signals forwarding
	go func() {
		for sig := range commandSig {
			if sig != syscall.SIGCHLD {
				// Forward signal to main process and all children
				syscall.Kill(-process.Process.Pid, sig.(syscall.Signal))
			}
		}
	}()

	err := process.Start()
	if err != nil {
		log.Fatal(
			"Failed starting command",
			zap.String("cmd", c.Command),
			zap.Error(err),
		)
	}

	err = process.Wait()
	msg := err.Error()
	// no *child processes is normal error* when we are pid 1
	if err != nil && !strings.Contains(msg, "no child processes") {
		if fatal {
			log.Fatal(
				"Process terminated",
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		} else {
			log.Warn(
				"Process terminated",
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		}
	} else {
		log.Info(
			"Process ended",
			zap.String("cmd", c.Command),
		)
	}
}

// RunBlockingNonFatal runs command in blocking mode
func (c *Command) RunBlockingNonFatal() {
	c.RunBlocking(false)
}

// Run executes process and redirects pipes
func (c *Command) Run() {
	go func(command, runAs string) {

		// Register chan to receive system signals
		commandSig := make(chan os.Signal, 1)
		defer close(commandSig)
		signal.Notify(commandSig)
		defer signal.Reset()
		wg.Add(1)
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
		log.Info("Starting", zap.Strings("args", args), zap.String("user", c.user))

		// start the process
		err := process.Start()
		if err != nil {
			log.Error(
				"Failed starting command",
				zap.String("cmd", command),
				zap.Error(err),
			)
		}

		// Goroutine for signals forwarding
		go func() {
			for sig := range commandSig {
				if sig != syscall.SIGCHLD {
					// Forward signal to main process and all children
					syscall.Kill(-process.Process.Pid, sig.(syscall.Signal))
				}
			}
		}()

		err = process.Wait()
		if err != nil {
			log.Fatal(
				"Process terminated",
				zap.String("cmd", c.Command),
				zap.Error(err),
			)
		}

	}(c.Command, c.RunAs)
}
