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
	"github.com/robfig/cron"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

var configLocation string
var loggerLocation string
var showVersion bool
var showDebug bool
var wg sync.WaitGroup
var log *zap.Logger

func init() {
	log, _ = zap.NewProduction()
	flag.StringVar(&configLocation, "config", "/provision/init.toml", "Location of the init file")
	flag.StringVar(&loggerLocation, "logger", "/var/www/mu-plugins/logger.sock", "Location of logger socket")
	flag.BoolVar(&showVersion, "version", false, "Show build time and version")
	flag.BoolVar(&showDebug, "debug", false, "Show detailed process traces")
}

func main() {
	flag.Parse()

	if showDebug {
		log, _ = zap.NewDevelopment()
	}

	log.Info(version.String())
	if showVersion {
		os.Exit(0)
	}
	cronRunner := cron.New()
	ctx, cancel := context.WithCancel(context.Background())
	pids := make(reap.PidCh, 1)
	errors := make(reap.ErrorCh, 1)
	done := make(chan struct{}, 1)
	mainHandler := make(chan os.Signal)
	signal.Notify(mainHandler, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		sig := <-mainHandler
		log.Info("Main interrupt done, canceling workers", zap.String("received", sig.String()))
		cancel()
		cronRunner.Stop()
		done <- struct{}{}
	}()

	go func() {
		for {
			select {
			case pid := <-pids:
				log.Debug("Reaped child process", zap.Int("pid", pid))
			case err := <-errors:
				log.Error("Error reaping child process", zap.Error(err))
			}
		}
	}()

	go reap.ReapChildren(pids, errors, done, nil)

	config := NewConfig(configLocation)
	if config.PreStart.Command != "" {
		config.PreStart.RunBlocking(false, false, ctx)
	}

	for _, job := range config.Cron {
		cj := Cron{}
		err := copier.Copy(&cj, &job)
		if err != nil {
			log.Fatal("Failed", zap.Error(err))
		}
		if cj.Enabled.IsActive() && !cj.Enabled.IsTrue() {
			log.Info(
				"Skipping cron",
				zap.String("cmd", cj.Command.Command),
				zap.Bool("enabled", cj.Enabled.IsTrue()),
				zap.String("schedule", cj.Schedule),
			)
			continue
		}
		log.Info(
			"Scheduling cron",
			zap.String("cmd", cj.Command.Command),
			zap.String("schedule", cj.Schedule),
			zap.Bool("SafeEnv", cj.SafeEnv),
		)
		if cj.SafeEnv {
			if err := cronRunner.AddFunc(cj.Schedule, func() { cj.RunBlockingNonFatalSafeEnv(ctx) }); err != nil {
				log.Fatal("Adding cron entry failed", zap.Error(err))
			}
		} else {
			if err := cronRunner.AddFunc(cj.Schedule, func() { cj.RunBlockingNonFatal(ctx) }); err != nil {
				log.Fatal("Adding cron entry failed", zap.Error(err))
			}
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

	for idx := range config.Process {
		proc := config.Process[idx]
		log.Info(
			"Command enabler",
			zap.Bool("isActive", proc.Enabled.IsActive()),
			zap.String("operator", proc.Enabled.Operator),
		)
		if proc.Enabled.IsActive() && !proc.Enabled.IsTrue() {
			log.Info(
				"Command not enabled",
				zap.String("cmd", proc.Command),
			)
			continue
		}
		if proc.Template.Enabled() {
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

		proc.Run(ctx, false)
	}

	if config.PostStart.Command != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			config.PostStart.RunBlocking(false, false, ctx)
		}()
	}

	log.Info("Staring logger socket", zap.String("location", loggerLocation))
	unixlog := NewUnixLogger(loggerLocation)
	go unixlog.Serve()
	log.Info("Event loop ready")

	wg.Wait()
	os.Exit(0)
}
