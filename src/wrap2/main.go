package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"gopkg.in/mcuadros/go-syslog.v2"
)

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Println(string(bytes))
}

var wrapCommand string
var wrapCommand2 string
var syslogBind string

func init() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))
	flag.StringVar(&wrapCommand, "cmd", "", "Command to wrap")
	flag.StringVar(&wrapCommand2, "cmd2", "", "Command to wrap")
	flag.StringVar(&syslogBind, "syslog", "127.0.0.1:514", "Address of internal syslog UDP server")
}

func safeSplit(s string) []string {
	split := strings.Split(s, " ")

	var result []string
	var inquote string
	var block string
	for _, i := range split {
		if inquote == "" {
			if strings.HasPrefix(i, "'") || strings.HasPrefix(i, "\"") {
				inquote = string(i[0])
				block = strings.TrimPrefix(i, inquote) + " "
			} else {
				result = append(result, i)
			}
		} else {
			if !strings.HasSuffix(i, inquote) {
				block += i + " "
			} else {
				block += strings.TrimSuffix(i, inquote)
				inquote = ""
				result = append(result, block)
				block = ""
			}
		}
	}
	return result
}

func run(command string) {

	cwd, err := os.Getwd()
	if err != nil {
		log.Panic(err)
	}

	pa := os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Dir:   cwd,
	}

	args := safeSplit(command)
	app := args[0]
	if !filepath.IsAbs(app) {
		app, err = exec.LookPath(app)
		if err != nil {
			log.Panic(err)
		}
	}
	proc, err := os.StartProcess(app, args[1:], &pa)
	if err != nil {
		log.Panic(err)
	}

	_, err = proc.Wait()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	os.Exit(1)
}

func parseSyslog(channel syslog.LogPartsChannel) {
	for logParts := range channel {
		if str, ok := logParts["content"].(string); ok {
			log.Println(str)
		}
	}
}

func main() {
	flag.Parse()
	if wrapCommand != "" {
		channel := make(syslog.LogPartsChannel)
		syslogServer := syslog.NewServer()
		syslogServer.SetFormat(syslog.RFC3164)
		syslogServer.SetHandler(syslog.NewChannelHandler(channel))
		syslogServer.ListenUDP(syslogBind)
		syslogServer.Boot()
		log.Println("Syslog server started on UDP:", syslogBind)
		log.Println("Starting command:", wrapCommand)
		go parseSyslog(channel)
		go run(wrapCommand)
		go run(wrapCommand2)

		// Handle SIGINT and SIGTERM.
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch //block

	} else {
		log.Fatalln("Wrap command")
	}
}
