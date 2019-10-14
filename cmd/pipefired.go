package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/masenocturnal/pipefire/internal/config"
	"github.com/masenocturnal/pipefire/internal/sftp"
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

	log.Print("- - - - - - - - - - - - - - -")
	log.Print("daemon started")
	conf, err := config.ReadApplicationConfig("pipefired")
	_ = conf
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			log.Println("Unable to find a configuration file")
		} else {
			// Config file was found but another error was produced
			log.Print("Encountered error: " + err.Error())
		}
	}
	initLogging(conf.LogLevel)
	err = executePipelines(conf)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Print("Flow done")
	}
}

func initLogging(lvl string) {
	log.SetFormatter(&log.JSONFormatter{})
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

func executePipelines(conf *config.HostConfig) error {
	// A common pattern is to re-use fields between logging statements by re-using
	// the logrus.Entry returned from WithFields()
	contextLogger := log.WithFields(log.Fields{
		"correlationId": uuid.New().String(),
	})

	contextLogger.Info("Starting Job")

	endPoint := conf.Sftp["connection1"]
	sftp, err := sftp.NewConnection("connection1", endPoint)
	if err != nil {
		contextLogger.Error(err.Error())
		return err
	}
	defer sftp.Close()

	// Get Remote Files
	foo, err := sftp.GetFile("/home/ubuntu/test.tar.bz2", "/tmp/")
	if err != nil {
		return err
	}

	result, _ := json.MarshalIndent(foo, "", " ")
	fmt.Println(string(result))

	confirmations, errors := sftp.SendDir("/home/andmas/tmp/RefundFiles", "/home/ubuntu/tmp")
	if errors != nil {
		// show all errors
		for temp := errors.Front(); temp != nil; temp = temp.Next() {
			fmt.Println(temp.Value)
		}
	}

	// show success
	for temp := confirmations.Front(); temp != nil; temp = temp.Next() {
		result, _ := json.MarshalIndent(temp.Value, "", " ")
		fmt.Println(string(result))

	}

	if err != nil {
		contextLogger.Error(err.Error())
		return err
	}
	//result, _ := json.MarshalIndent(confirmation, "", " ")

	//fmt.Print(string(result))

	return nil
}
