package main

import (
	"encarno/pkg/core"
	"encarno/pkg/http"
	"encarno/pkg/scenario"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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
	log.Infof("Loading config file: %s", path)
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	cfg := core.Configuration{
		Protocol: core.ProtoConf{
			MaxConnections: 1,
			Timeout:        1 * time.Second,
		},
	}
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
			log.Infof("Got signal %d: %v", s, s)
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
	status := new(core.Status)
	status.Start()

	output := core.NewMultiOutput(config.Output)
	output.Start(config.Output)
	defer output.Close()

	nibMaker := NewNibMaker(config.Protocol)

	controller = NewWorkload(config.Workers, config.Input, nibMaker, output, status)
	controller.Run()
}

func NewNibMaker(protocol core.ProtoConf) core.NibMaker {
	log.Infof("Client protocol is: %s", protocol.Driver)
	switch protocol.Driver {
	case "dummy":
		return func() core.Nib {
			return &core.DummyNib{}
		}
	case "http":
		pool := http.NewConnectionPool(protocol.MaxConnections, protocol.Timeout, protocol)

		return func() core.Nib {
			return &http.Nib{
				ConnPool: pool,
			}
		}
	default:
		panic(fmt.Sprintf("Unsupported protocol driver: %v", protocol.Driver))
	}
}

func NewWorkload(workersConf core.WorkerConf, inputConfig core.InputConf, nibMaker core.NibMaker, output core.Output, status *core.Status) core.WorkerSpawner {
	base := core.NewBaseWorkload(nibMaker, output, inputConfig, workersConf, status)
	switch workersConf.Mode {
	case "open":
		return scenario.NewOpenWorkload(workersConf, base)
	case "closed":
		return scenario.NewClosedWorkload(inputConfig, base)
	default:
		panic(fmt.Sprintf("Unsupported workers mode: %s", workersConf.Mode))
	}
}
