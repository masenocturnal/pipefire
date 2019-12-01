package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/masenocturnal/pipefire/internal/config"
	"github.com/masenocturnal/pipefire/pipelines/directdebit"
	"github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const version string = "0.9.10"

func main() {

	cntxt := &daemon.Context{
		PidFileName: "pipfire.pid",
		PidFilePerm: 0644,
		LogFileName: "pipefire.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"[go-daemon sample]"},
	}
	_ = cntxt

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
	hostConfig, err := config.ReadApplicationConfig("pipefired")
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Println("Unable to find a configuration file")
		} else {
			// Config file was found but another error was produced
			log.Print("Encountered error: " + err.Error())
		}
		os.Exit(1)
	}
	initLogging(hostConfig.LogLevel)

	// @todo make this dynamic
	ddConfig := hostConfig.Pipelines.DirectDebit

	// create the dd pipeline
	directDebitPipeline, err := directdebit.New(&ddConfig)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	for {
		listenerError := make(chan error)

		go directDebitPipeline.StartListener(listenerError)
		err := <-listenerError

		log.Warningf("RabbitMQ Reconnect Required: %s", err)
		log.Debugf("No of goroutines %d", runtime.NumGoroutine())

		time.Sleep(2 * time.Second)

	}

}

func initLogging(lvl string) {
	//log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{})

	log.SetOutput(os.Stdout)

	lvl = strings.ToLower(lvl)

	switch lvl {
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
