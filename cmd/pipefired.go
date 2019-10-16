package main

import (
	"os"
	"strings"

	"github.com/masenocturnal/pipefire/internal/config"
	"github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {

	cntxt := &daemon.Context{
		PidFileName: "sample.pid",
		PidFilePerm: 0644,
		LogFileName: "sample.log",
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

	log.Info("PipeFire Daemon Started")
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

	err = directDebitPipeline(hostConfig)
	// err = executePipelines(conf)
	if err != nil {
		log.Error(err.Error())
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
		log.SetLevel(log.WarnLevel)
		break
	}
}
