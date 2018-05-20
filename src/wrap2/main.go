package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"
)

var configLocation string
var wg sync.WaitGroup
var log *zap.Logger

func init() {
	log, _ = zap.NewProduction()
	defer log.Sync()
	flag.StringVar(&configLocation, "config", "/provision/init.toml", "Location of the init file")
}

func main() {

	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())

	config := NewConfig(configLocation)
	if config.PreStart.Command != "" {
		config.PreStart.RunBlocking()
	}

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
		proc.Run(ctx, cancel)
	}

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch //block

	cancel()
	wg.Wait()
}
