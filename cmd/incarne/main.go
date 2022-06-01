package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"incarne/pkg/core"
	"incarne/pkg/http"
	"incarne/pkg/scenario"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var controller core.WorkerSpawner

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

	config := LoadConfig(flag.Arg(0))
	Run(config)
}

func LoadConfig(path string) core.Configuration {
	yamlFile, err := ioutil.ReadFile(path)
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

var alreadyHandlingSignal = false

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
			if !alreadyHandlingSignal {
				alreadyHandlingSignal = true

				if controller != nil {
					controller.Interrupt()
				}
			}
			os.Exit(2)
		}
	}()
}

func Run(config core.Configuration) {
	nibMaker := NewNibMaker(config.Protocol)
	spawner := NewSpawner(config.Workers, config.Input, nibMaker)

	var output core.Output
	output.Start(config.Output)

	spawner.Run()
}

func NewNibMaker(protocol core.ProtoConf) core.NibMaker {
	log.Infof("Client protocol is: %s", protocol.Driver)
	switch protocol.Driver {
	case "dummy":
		return func() core.Nib {
			return &core.DummyNib{}
		}
	case "http":
		httpConf := http.ParseHTTPConf(protocol)
		pool := &http.ConnPool{
			Idle:           make(map[string]http.ConnChan),
			MaxConnections: httpConf.MaxConnections,
			Timeout:        httpConf.Timeout * time.Second,
		}

		return func() core.Nib {
			return &http.Nib{
				ConnPool: pool,
			}
		}
	default:
		panic(fmt.Sprintf("Unsupported protocol driver: %v", protocol.Driver))
	}
}

func NewSpawner(workers core.WorkerConf, inputConfig core.InputConf, maker core.NibMaker) core.WorkerSpawner {
	switch workers.Mode {
	case "open":
		inputChannel := core.NewInput(inputConfig)
		workload := scenario.OpenWorkload{
			Status:     nil,
			MinWorkers: 0,
			MaxWorkers: 0,
			Input:      inputChannel,
		}
		return &workload
	case "closed":
		workload := scenario.ClosedWorkload{
			Scenario:    nil, // TODO
			InputConfig: inputConfig,
		}
		return &workload
	default:
		panic(fmt.Sprintf("Unsupported workers mode: %s", workers.Mode))
	}
}
