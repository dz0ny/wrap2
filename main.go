package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dz0ny/wrap2/version"
	"github.com/hashicorp/go-reap"

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

func main() {

	flag.Parse()
	log.Info(version.String())
	if showVersion {
		os.Exit(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	pids := make(reap.PidCh, 1)
	errors := make(reap.ErrorCh, 1)
	done := make(chan struct{})
	mainHandler := make(chan os.Signal, 1)
	signal.Notify(mainHandler, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGKILL)

	go func() {
		sig := <-mainHandler
		log.Info("Main interrupt done, canceling workers", zap.String("received", sig.String()))
		cancel()
		done <- struct{}{}
	}()

	go func() {
		for {
			select {
			case pid := <-pids:
				log.Info("Reaped child process", zap.Int("pid", pid))
			case err := <-errors:
				log.Error("Error reaping child process", zap.Error(err))
			case <-done:
				return
			}
		}
	}()

	go reap.ReapChildren(pids, errors, done, nil)

	cronRunner := cron.New()
	config := NewConfig(configLocation)
	if config.PreStart.Command != "" {
		config.PreStart.RunBlocking(false, ctx)
	}

	for _, job := range config.Cron {
		cj := Cron{}
		copier.Copy(&cj, &job)
		log.Info(
			"Scheduling cron",
			zap.String("cmd", cj.Command.Command),
			zap.String("schedule", cj.Schedule),
		)
		if err := cronRunner.AddFunc(cj.Schedule, func() { cj.RunBlockingNonFatal(ctx) }); err != nil {
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

		proc.Run(ctx)
	}

	if config.PostStart.Command != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			config.PostStart.RunBlocking(false, ctx)
		}()
	}

	log.Info("Staring logger socket", zap.String("location", loggerLocation))
	unixlog := NewUnixLogger(loggerLocation)
	go unixlog.Serve()
	log.Info("Event loop ready")

	wg.Wait()
	os.Exit(0)
}
