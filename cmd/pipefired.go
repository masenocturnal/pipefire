package main

import (
	"os"
	"strings"

	uuid "github.com/google/uuid"
	"github.com/masenocturnal/pipefire/internal/config"
	"github.com/masenocturnal/pipefire/pipelines/directdebit"
	"github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const version string = "0.9.1"

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

	// d, err := cntxt.Reborn()
	// if err != nil {
	// 	log.Fatal("Unable to run: ", err)
	// }
	// if d != nil {
	// 	return
	// }
	// defer cntxt.Release()

	log.Infof("PipeFire Daemon Started. Version : %s ", version)
	hostConfig, err := config.ReadApplicationConfig("pipefired")

	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Println("Unable to find a configuration file")
		} else {
			// Config file was found but another error was produced
			log.Print("Encountered error: " + err.Error())
		}
	}
	initLogging(hostConfig.LogLevel)
	correlationID := uuid.New()

	logEntry := log.WithFields(log.Fields{
		"correlationId": correlationID,
	})

	// create the dd pipeline
	directDebitPipeline := directdebit.New(correlationID.String(), logEntry)

	// @todo make this dynamic
	ddConfig := hostConfig.Pipelines.DirectDebit

	// @todo load and execute pipelines concurrently
	// execute pipeline
	pipelineErrors := directDebitPipeline.Execute(&ddConfig)

	// err = executePipelines(conf)
	if pipelineErrors != nil && len(pipelineErrors) > 0 {
		for _, err := range pipelineErrors {
			log.Error(err.Error())
		}
	} else {
		log.Info("Direct Debit Pipeline Done")
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
