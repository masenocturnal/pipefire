package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"plugin"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/masenocturnal/pipefire/internal/common_interfaces"
	"github.com/masenocturnal/pipefire/internal/config"
	"github.com/masenocturnal/pipefire/internal/mq"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const version string = "0.10.00"

func main() {

	log.Infof("PipeFire Daemon Started. Version : %s ", version)

	// create the channel to handle the OS Signal
	signalChannel := make(chan os.Signal, 1)

	// ask to be notified of signals. @todo we actually need to deal with this differently
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGHUP)
	go executePipelines()
	<-signalChannel
	fmt.Println("Pipefire Shutting Down")
	//
	os.Exit(0)

}

func executePipelines() {
	hostConfig, err := config.ReadApplicationConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Fatal("Unable to find a configuration file")
		} else {
			// Config file was found but another error was produced
			log.Fatal("Encountered error: " + err.Error())
		}
		os.Exit(1)
	}

	initLogging(hostConfig.GetString("loglevel"))

	c := &config.HostConfig{}
	err = hostConfig.Unmarshal(c)
	if err != nil {
		log.Fatal(err.Error())
	}

	selectedConfig := hostConfig.ConfigFileUsed()
	selectedDir := path.Dir(selectedConfig)
	log.Infof("Using %s", selectedConfig)

	// load configuration
	for pipelineName := range c.Pipelines {

		file := c.Pipelines[pipelineName]
		x := path.Join(selectedDir, file)
		log.Infof("looking for %s", x)
		jsonText, err := ioutil.ReadFile(x)
		if err != nil {
			log.Warningf("Unable to read file %s for pipeline %s", file, pipelineName)
			continue
		}

		pipelineConfig := &config.PipelineConfig{}

		if err := json.Unmarshal(jsonText, pipelineConfig); err != nil {
			log.Errorf("Unable to load config for pipeline: %s ", pipelineName)
			log.Error(err.Error())
			os.Exit(1)
		} else {
			log.Debugf("Config %s loaded", x)
		}

		// load plugin
		p, err := plugin.Open(c.PluginDir + pipelineName + ".so")
		if err != nil {
			log.Errorf("Unable to load plugin for %s", pipelineName)
			log.Error(err.Error())
			continue
		}

		// look for the function that defines the version string
		GetVersion, err := p.Lookup("GetVersion")
		if err != nil {
			log.Errorf("Unable to find the plugin version for %s", pipelineName)
			log.Error(err.Error())
			continue
		} else {
			version := GetVersion.(func() string)()
			log.Infof("Pipefire Pipeline Plugin: %s version %s has been loaded successfully", pipelineName, version)
		}

		// look for the function that defines the version string
		New, err := p.Lookup("New")
		if err != nil {
			log.Errorf("Unable to find the New function for %s", pipelineName)
			log.Error(err.Error())
			continue
		} else {
			pipeline, err := New.(func(*config.PipelineConfig) (interface{}, error))(pipelineConfig)

			if err != nil {
				log.Error(err.Error())

				// we are really expecting these to establish without a hitch
				// if it's not in a good state, we would rather bail as when we start listenging for triggers
				// we're really expecting to walk away and for the pipelines to execute correctly
				os.Exit(1)
			}

			for {

				log.Debugf("No of goroutines %d", runtime.NumGoroutine())
				listenerError := make(chan error)

				p := pipeline.(common_interfaces.PipelineInterface)
				go p.StartListener(listenerError)
				err := <-listenerError

				log.Warningf("RabbitMQ Reconnect Required: %s", err)
				time.Sleep(2 * time.Second)
			}

		}
	}
}

type CustomPipeline struct {
	Log            *log.Entry
	correlationID  string
	consumer       *mq.MessageConsumer
	pipelineConfig *config.PipelineConfig
}

func initLogging(lvl string) {
	//log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{})

	log.SetOutput(os.Stdout)

	lvl = strings.ToLower(lvl)

	switch lvl {
	case "trace":
		log.SetLevel(log.TraceLevel)
		break
	case "debug":
		log.SetLevel(log.DebugLevel)
		break
	case "warning":
		log.SetLevel(log.WarnLevel)
		break
	case "information":
		log.SetLevel(log.InfoLevel)
		break
	}
}
