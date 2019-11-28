package main

import (
	"bytes"
	"fmt"
	"net"
	"os"

	"go.uber.org/zap"
)

// UnixLogger handles the external logging messages
type UnixLogger struct {
	path string
}

// NewUnixLogger creates new unix logger server in path
func NewUnixLogger(path string) *UnixLogger {
	return &UnixLogger{path}
}

// Serve starts unixpath to stdout logger
func (ul *UnixLogger) Serve() {

	os.Remove(ul.path)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: ul.path, Net: "unix"})
	if err != nil {
		log.Fatal("ListenUnix failed", zap.Error(err))
	}
	err = os.Chmod(ul.path, 0666)
	if err != nil {
		log.Fatal("Chmod failed", zap.Error(err))
	}
	defer os.Remove(ul.path)

	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			log.Warn("AcceptUnix failed", zap.Error(err))
			return
		}

		// growing buffer, starting with 1024 bytes
		buf := bytes.NewBuffer(make([]byte, 0, 1024))
		_, err = buf.ReadFrom(conn)
		if err != nil {
			log.Warn("ReadFrom failed", zap.Error(err))
			conn.Close()
			return
		}
		fmt.Println(buf.String())
		conn.Close()
	}
}
