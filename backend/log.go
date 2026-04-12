// log.go
package main

import (
	"bytes"
	"io"
	"log"
	"log/syslog"
	"os"
	"time"
)

func initTZ() {
	if os.Getenv("TZ") == "" {
		if raw, err := os.ReadFile("/etc/TZ"); err == nil {
			tz := string(bytes.TrimSpace(raw))
			if tz != "" {
				os.Setenv("TZ", tz)
			}
		}
	}
	if loc, err := time.LoadLocation(os.Getenv("TZ")); err == nil {
		time.Local = loc
	}
}

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
