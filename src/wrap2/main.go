package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"wrap2/version"

	"github.com/robfig/cron"
	"go.uber.org/zap"
)

var configLocation string
var showVersion bool
var wg sync.WaitGroup
var log *zap.Logger

func init() {
	log = zap.NewExample()
	defer log.Sync()
	flag.StringVar(&configLocation, "config", "/provision/init.toml", "Location of the init file")
	flag.BoolVar(&showVersion, "version", false, "Show build time and version")
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

	for _, cj := range config.Cron {
		log.Info(
			"Scheduling cron",
			zap.String("cmd", cj.Command.Command),
			zap.String("schedule", cj.Schedule),
		)
		cronRunner.AddFunc(cj.Schedule, cj.RunBlocking)
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

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch //block

	cancel()
	wg.Wait()
}
