// Copyright 2020 Richard Maynard (richard.maynard@gmail.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"io/ioutil"
	"log/syslog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
	flag "github.com/spf13/pflag"
)

var (
	listenIP      string
	listenPort    string
	checkURI      string
	checkMatch    string
	syslogEnabled bool
	sysLogAddr    string
	sysLogProto   string
	logLevel      string
	logger        *log.Logger
)

func init() {
	// logger.SetReportCaller(true)
	logger = log.New()

	flag.StringVar(&listenIP, "bind-address", "0.0.0.0", "IP address to bind to")
	flag.StringVar(&listenPort, "bind-port", "1580", "Port to listen on")
	flag.StringVar(&checkURI, "check-uri", "http://localhost:8080/healthz", "URI to check")
	flag.StringVar(&checkMatch, "check-match", "(?i)^ok\\b", "golang regex to match (https://golang.org/pkg/regexp/syntax/, escape \\'s\n\tuse '.*' to match anything and rely only on status code\n\tdefault pattern is case insentive, looking for ok followed by a word boundry")
	flag.BoolVar(&syslogEnabled, "syslog-enable", false, "Enable syslog messages")
	flag.StringVar(&sysLogAddr, "syslog-addr", "", "Syslog address and port, ex: 127.0.0.1:514 (blank for local syslog socket)")
	flag.StringVar(&sysLogProto, "syslog-proto", "", "protocol to use for syslog (blank for local syslog socket)")
	flag.StringVar(&logLevel, "log-level", "info", "logging level (trace, debug, info, warning, fatal")
	flag.Parse()
}

func logSetup(level string) {
	var syslogLevel syslog.Priority

	switch strings.ToLower(level) {
	case "trace":
		logger.SetLevel(log.TraceLevel)
		syslogLevel = syslog.LOG_DEBUG
	case "debug":
		logger.SetLevel(log.DebugLevel)
		syslogLevel = syslog.LOG_DEBUG
	case "warn", "warning", "notice":
		logger.SetLevel(log.WarnLevel)
		syslogLevel = syslog.LOG_WARNING
	case "err", "error":
		logger.SetLevel(log.ErrorLevel)
		syslogLevel = syslog.LOG_ERR
	case "crit", "fatal", "emerg", "panic", "alert":
		logger.SetLevel(log.PanicLevel)
		syslogLevel = syslog.LOG_CRIT
	default:
		logger.SetLevel(log.InfoLevel)
		syslogLevel = syslog.LOG_INFO
	}

	if syslogEnabled {
		logger.Out = ioutil.Discard
		logger.Debug("syslog enabled, changing logformat, and disabling stdout")
		logger.Formatter = &log.TextFormatter{DisableColors: true, DisableTimestamp: true}
		hook, err := logrus_syslog.NewSyslogHook(sysLogProto, sysLogAddr, syslogLevel, "")
		if err != nil {
			logger.Fatalf("error initializing syslog: %s", err.Error())
		} else {
			logger.Trace("adding syslog hook")
			logger.AddHook(hook)
		}
	}

}

// checkHealthControl manages the loop of health checks
func checkHealthControl(statusChan chan bool, controlChan chan bool) {
	firstRun := true
	re := regexp.MustCompile(checkMatch)
	for {
		logger.Trace("entering health check control loop")
		select {
		case <-controlChan:
			logger.Debug("recieved shutdown signal, returning from health check control loop")
			return
		default:
		}
		if !firstRun {
			logger.Trace("this is not first run, waiting 5 seconds before checking health again")
			time.Sleep(5 * time.Second)
		}
		logger.Trace("executing health check")
		checkHealth(statusChan, checkURI, re)
		logger.Trace("returned from health check")
		firstRun = false
	}
}

// checkHealth performs a request against a remote server, and responds to a channel with true/false
// indicating the health of the remote check
func checkHealth(c chan bool, uri string, checkBody *regexp.Regexp) {
	logger.WithFields(log.Fields{
		"URI":       uri,
		"CheckBody": checkBody}).Trace("executing health check")

	logger.Trace("http.Get start")
	resp, err := http.Get(uri)
	logger.Trace("http.Get done")
	if err != nil {
		logger.WithError(err).Warn("health check connection failed")
		c <- false
		return
	}

	logger.Trace("defer body close")
	defer resp.Body.Close()

	logger.Trace("check status code")
	if !(resp.StatusCode == http.StatusOK) {
		logger.WithFields(log.Fields{"StatusCode": resp.StatusCode}).Warn("health check failed status code failed")
		c <- false
		return
	}

	logger.Trace("read body content")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c <- false
		return
	}

	logger.Trace("compare body content")
	if checkBody.FindString(string(body)) == "" {
		logger.WithFields(log.Fields{"checkMatch": checkMatch, "content": string(body)}).Warnf("health check failed checkMatch not in content")
		c <- false
		logger.Trace("submitted false to chan")
		return
	}

	logger.Trace("everything went well")
	c <- true
}

func main() {
	logSetup(logLevel)
	statusChan := make(chan bool)
	controlChan := make(chan bool)
	sigChan := make(chan os.Signal)
	var status bool

	// create an echo server
	logger.Trace("creating echo instance")
	p := NewEcho(listenIP, listenPort)

	// start health check thread
	logger.Trace("starting health check control loop")
	go checkHealthControl(statusChan, controlChan)

	// start signal handler
	logger.Trace("registering signal handlers")
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func(*Echo, chan bool) {
		logger.Trace("starting wait for message on signal channel")
		sig := <-sigChan
		logger.WithFields(log.Fields{"signal": sig}).Info("recieved signal, shutting down")
		// immediately close status and control channels so they don't try and start
		// a new echo server
		logger.Trace("closing control/status channels")
		close(statusChan)
		logger.Trace("initiating echo server shutdown")
		p = p.Down()
		logger.Info("graceful shutdown complete")
		close(controlChan)
	}(p, controlChan)

	// main loop, read status updates and handle echo service properly
	for {
		logger.Trace("waiting for message on status check channel")
		select {
		case <-controlChan:
			logger.Trace("recieved shutdown signal, exiting from main loop")
			return
		default:
		}
		status = <-statusChan
		if status {
			logger.Trace("message was true, calling up on echo server")
			p = p.Up()
		} else {
			logger.Trace("message was false, calling down on echo server")
			p = p.Down()
		}
	}
}
