package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/masenocturnal/pipefire/internal/config"
	"github.com/masenocturnal/pipefire/internal/sftp"
	"github.com/masenocturnal/pipefire/internal/crypto"
	"github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/openpgp"
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

	log.Info("Starting Pipeline")
	err = executePipelines(conf)
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Info("Pipeline Done")
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

func encryptPipeline() {
	var pubKey string
	log.Println("Public key:", pubKey)

	provider := 

	// Read in public key
	recipient, err := KeyFromFile(pubKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	f, err := os.Open(fileToEnc)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	dst, err := os.Create(fileToEnc + ".gpg")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer dst.Close()
	encrypt([]*openpgp.Entity{recipient}, nil, f, dst)
}

func executePipelines(conf *config.HostConfig) error {
	correlationID := uuid.New().String()
	// A common pattern is to re-use fields between logging statements by re-using
	// the logrus.Entry returned from WithFields()
	contextLogger := log.WithFields(log.Fields{
		"correlationId": correlationID,
	})

	contextLogger.Info("Starting Pipeline")

	endPoint := conf.Sftp["connection1"]
	sftp, err := sftp.NewConnection("connection1", endPoint, correlationID)
	if err != nil {
		contextLogger.Error(err.Error())
		return err
	}
	defer sftp.Close()

	// // Get Remote File
	// foo, err := sftp.GetFile("/home/am/positivessl.zip", "/tmp/")
	// if err != nil {
	// 	return err
	// }

	// Get Remote Dir
	status, errors := sftp.GetDir("/home/am/nocturnal.net.au", "/tmp/foobar")
	if errors != nil {
		// show all errors
		for temp := errors.Front(); temp != nil; temp = temp.Next() {
			fmt.Println(temp.Value)
		}
	}

	if errors.Len() == 0 {
		if err := sftp.CleanDir("/home/am/nocturnal.net.au"); err != nil {
			contextLogger.Error(err.Error())
			return err
		}
	}

	// result, _ := json.MarshalIndent(foo, "", " ")
	// fmt.Println(string(result))

	// confirmations, errors := sftp.SendDir("/home/andmas/tmp/RefundFiles", "/home/ubuntu/tmp")
	// if errors != nil {
	// 	// show all errors
	// 	for temp := errors.Front(); temp != nil; temp = temp.Next() {
	// 		fmt.Println(temp.Value)
	// 	}
	// }

	// // show success
	// @todo loop recursive

	for temp := status.Front(); temp != nil; temp = temp.Next() {
		result, _ := json.MarshalIndent(temp.Value, "", " ")
		log.Debug(string(result))
	}

	// if err != nil {
	// 	contextLogger.Error(err.Error())
	// 	return err
	// }

	return nil
}
