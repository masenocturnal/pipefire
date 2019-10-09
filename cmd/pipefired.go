package main

import (
	"fmt"
	"github.com/sevlyar/go-daemon"
	"github.com/masenocturnal/pipefire/internal/config"
)

func main() {

	// do we need this with systemd ?
	cntxt := &daemon.Context{
		PidFileName: "pipefired.pid",
		PidFilePerm: 0644,
		LogFileName: "pipefired.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"[background false]"},
	}

	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Unable to run: ", err)
	}
	if d != nil {
		return
	}
	defer cntxt.Release()

	log.Print("- - - - - - - - - - - - - - -")
	log.Print("daemon started")

}

func loadConfig()
{
	configurationFile := "/etc/pipefire/pipefired.json"
	if os.Getenv("DEV") != "" {
		// default configuration file for dev
		configurationFile = "../configs/pipefired.json"
	}

	// load config
	appConfig, err := config.ReadApplicationConfig(configurationFile)
}

func run() {

}

