package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dz0ny/wrap2/version"

	"github.com/jinzhu/copier"
	"github.com/robfig/cron"
	"go.uber.org/zap"
)

var configLocation string
var loggerLocation string
var showVersion bool
var wg sync.WaitGroup
var log *zap.Logger

func init() {
	log = zap.NewExample()
	flag.StringVar(&configLocation, "config", "/provision/init.toml", "Location of the init file")
	flag.StringVar(&loggerLocation, "logger", "/var/www/mu-plugins/logger.sock", "Location of logger socket")
	flag.BoolVar(&showVersion, "version", false, "Show build time and version")
}

// From https://github.com/ramr/go-reaper/blob/master/reaper.go
func reapLoop() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGCHLD)
	for range c {
		reapChildren()
	}
}

func reapChildren() {
	for {
		var (
			ws  syscall.WaitStatus
			pid int
			err error
		)
		for {
			pid, err = syscall.Wait4(-1, &ws, 0, nil)
			if err != syscall.EINTR {
				break
			}
		}
		if err == syscall.ECHILD {
			return // done
		}
		log.Info("reaped child process", zap.Int("pid", pid), zap.Int32("ws", int32(ws)))
	}
}

func main() {

	flag.Parse()

	if showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	cronRunner := cron.New()
	ctx, cancel := context.WithCancel(context.Background())

	config := NewConfig(configLocation)
	if config.PreStart.Command != "" {
		config.PreStart.RunBlocking()
	}

	for _, job := range config.Cron {
		cj := Cron{}
		copier.Copy(&cj, &job)
		log.Info(
			"Scheduling cron",
			zap.String("cmd", cj.Command.Command),
			zap.String("schedule", cj.Schedule),
		)
		if err := cronRunner.AddFunc(cj.Schedule, cj.RunBlocking); err != nil {
			log.Fatal("Adding cron entry failed", zap.Error(err))
		}
	}
	cronRunner.Start()

	for _, proc := range config.Process {
		if err := proc.Template.Process(); err != nil {
			log.Fatal(
				"process config failed",
				zap.String("cmd", proc.Command),
				zap.Error(err),
			)
		}
	}

	for _, proc := range config.Process {
		if proc.Template.Source != "" && proc.Template.Target != "" {
			log.Info(
				"Parsing",
				zap.String("src", proc.Template.Source),
				zap.String("dst", proc.Template.Target),
			)
			if err := proc.Template.Process(); err != nil {
				log.Fatal(
					"template parsing failed",
					zap.String("src", proc.Template.Source),
					zap.String("dst", proc.Template.Target),
					zap.Error(err),
				)
			}
		}
		proc.Run(ctx, cancel)
	}

	if config.PostStart.Command != "" {
		config.PostStart.RunBlocking()
	}

	unixlog := NewUnixLogger(loggerLocation)
	go unixlog.Serve()

	go reapLoop()

	wg.Wait()
	cancel()
}
