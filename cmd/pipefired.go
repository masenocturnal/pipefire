package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/masenocturnal/pipefire/internal/config"
	"github.com/masenocturnal/pipefire/pipelines/directdebit"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const version string = "0.9.11"

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
	// @todo shift this to the pipeline
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

	pipelineName := "directdebit"
	if c.Pipelines[pipelineName] != "" {
		file := c.Pipelines[pipelineName]
		x := path.Join(selectedDir, file)
		log.Infof("looking for %s", x)
		jsonText, err := ioutil.ReadFile(x)
		if err != nil {
			log.Warningf("Unable to read file %s for pipeline %s", file, pipelineName)
		}

		ddConfig := &directdebit.PipelineConfig{}
		if err := json.Unmarshal(jsonText, ddConfig); err != nil {
			log.Fatal("Unable to load config for directdebit pipeline")
		}

		// create the dd pipeline
		directDebitPipeline, err := directdebit.New(ddConfig)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}

		for {

			log.Debugf("No of goroutines %d", runtime.NumGoroutine())
			listenerError := make(chan error)

			go directDebitPipeline.StartListener(listenerError)
			err := <-listenerError

			log.Warningf("RabbitMQ Reconnect Required: %s", err)
			time.Sleep(2 * time.Second)
		}
	}

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
