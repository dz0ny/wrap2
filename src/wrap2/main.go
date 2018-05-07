package main

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

func main() {

	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())

	config := NewConfig(configLocation)

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
		if err := proc.Template.Process(); err != nil {
			log.Fatal(
				"template parsing failed",
				zap.String("src", proc.Template.Source),
				zap.String("dst", proc.Template.Target),
				zap.Error(err),
			)
		}

		go proc.Run(ctx, cancel)
	}

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch //block

	cancel()
	wg.Wait()
}
