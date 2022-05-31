package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"incarne/pkg/core"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

var controller *core.Controller

func main() {
	if os.Getenv("DEBUG") == "" {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	handleSignals()

	help := flag.Bool("help", false, "Show help")

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		fmt.Println("Missing configuration file path")
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	config := LoadConfig()
	controller.RunWithConfig(config)
}

func LoadConfig() core.Configuration {
	yamlFile, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		panic(err)
	}

	cfg := core.Configuration{}
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		panic(err)
	}
	return cfg
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

			if controller != nil {
				controller.Interrupt()
			}

			os.Exit(2)
			return
		}
	}()
}
