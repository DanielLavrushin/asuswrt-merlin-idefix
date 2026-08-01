// log.go
package main

import (
	"io"
	"log"
	"log/syslog"
	"os"
)

var Syslog *syslog.Writer

func setupLogging() {
	var err error
	Syslog, err = syslog.New(syslog.LOG_INFO|syslog.LOG_USER, "IDEFIX")
	if err != nil {
		log.Printf("could not open syslog, falling back to stderr: %v", err)
		return
	}

	log.SetFlags(0)
	log.SetOutput(io.MultiWriter(os.Stderr, Syslog))
}
