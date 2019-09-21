package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"

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

func cleanQuit(cancel context.CancelFunc) {
	// Signal zombie goroutine to stop
	// and wait for it to release waitgroup
	wg.Wait()
	os.Exit(0)
}

func main() {

	flag.Parse()

	if showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	cronRunner := cron.New()
	_, cancel := context.WithCancel(context.Background())

	config := NewConfig(configLocation)
	if config.PreStart.Command != "" {
		config.PreStart.RunBlocking(true)
	}

	for _, job := range config.Cron {
		cj := Cron{}
		copier.Copy(&cj, &job)
		log.Info(
			"Scheduling cron",
			zap.String("cmd", cj.Command.Command),
			zap.String("schedule", cj.Schedule),
		)
		if err := cronRunner.AddFunc(cj.Schedule, cj.RunBlockingNonFatal); err != nil {
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

		proc.Run()
	}

	if config.PostStart.Command != "" {
		config.PostStart.RunBlocking(true)
	}

	unixlog := NewUnixLogger(loggerLocation)
	go unixlog.Serve()
	go Reap()
	// Wait removeZombies goroutine
	cleanQuit(cancel)
}
