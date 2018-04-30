package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/mcuadros/go-syslog.v2"
)

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Println(string(bytes))
}

var wrapCommand string
var wrapCommand2 string
var wg sync.WaitGroup

func init() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))
	flag.StringVar(&wrapCommand, "cmd", "", "Command to wrap")
	flag.StringVar(&wrapCommand2, "cmd2", "", "Command to wrap")
}

func runCmd(ctx context.Context, cancel context.CancelFunc, cmd string) {
	defer wg.Done()
	args := strings.Split(cmd, " ")
	process := exec.Command(args[0], args[1:]...)
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr

	// start the process
	err := process.Start()
	if err != nil {
		log.Fatalf("Error starting command: `%s` - %s\n", cmd, err)
	}

	// Setup signaling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case sig := <-sigs:
			log.Printf("%s received signal: %s\n", args[0], sig)
			signalProcessWithTimeout(process, sig)
			cancel()
		case <-ctx.Done():
			// exit when context is done
		}
	}()

	err = process.Wait()
	cancel()

	if err == nil {
		log.Println("Command finished successfully.")
	} else {
		log.Printf("Command exited with error: %s\n", err)
		// OPTIMIZE: This could be cleaner
		os.Exit(err.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus())
	}

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
		log.Println("Killing command due to timeout.")
		process.Process.Kill()
	}
}

func parseSyslog(channel syslog.LogPartsChannel) {
	for logParts := range channel {
		fmt.Println(logParts)
	}
}

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	channel := make(syslog.LogPartsChannel)
	syslogServer := syslog.NewServer()
	syslogServer.SetFormat(syslog.RFC5424)
	syslogServer.SetHandler(syslog.NewChannelHandler(channel))
	syslogServer.ListenUDP("0.0.0.0:514")
	syslogServer.ListenTCP("0.0.0.0:514")
	syslogServer.Boot()
	log.Println("Syslog server started")
	go parseSyslog(channel)
	go runCmd(ctx, cancel, wrapCommand)
	go runCmd(ctx, cancel, wrapCommand2)
	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch //block
	cancel()
	wg.Wait()
}
