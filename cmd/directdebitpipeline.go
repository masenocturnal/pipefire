package main

import (
	"github.com/google/uuid"
	"github.com/masenocturnal/pipefire/internal/config"
	log "github.com/sirupsen/logrus"
)

func directDebitPipeline(hostConfig *config.HostConfig) (err error) {
	correlationID := uuid.New().String()

	contextLogger := log.WithFields(log.Fields{
		"correlationId": correlationID,
	})
	contextLogger.Info("Starting Direct Debit Pipeline")

	/*
		connection1Endpoint := hostConfig.Sftp["connection1"]
		if err = sftpFromTask(connection1Endpoint, correlationID); err != nil {
			return
		}
	*/

	encryptForANZ := hostConfig.Crypto["encryptforanz"]
	if err = encryptTask(encryptForANZ, correlationID); err != nil {
		return
	}
	/*
		connection2Endpoint := hostConfig.Sftp["connection2"]
		if err = sftpToTask(connection2Endpoint, correlationID); err != nil {
			return
		}

		connection3Endpoint := hostConfig.Sftp["connection3"]
		if err = sftpToTask(connection3Endpoint, correlationID); err != nil {
			return
		}
	*/
	return
}
