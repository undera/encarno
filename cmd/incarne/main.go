package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if os.Getenv("DEBUG") == "" {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	handleSignals()
	// consume YAML configuration
	// nib-specific configuration
}

func handleSignals() {
	signalChanel := make(chan os.Signal, 1)
	signal.Notify(signalChanel,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		select {
		case s := <-signalChanel:
			log.Infof("Got signal: %v", s.String())
			//application.Stop()
			os.Exit(2)
			return
		}
	}()
}
