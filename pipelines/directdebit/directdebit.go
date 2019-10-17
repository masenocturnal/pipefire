package directdebit

import (
	log "github.com/sirupsen/logrus"
)

// Pipeline is an implementation of a pipeline
type Pipeline interface {
	Execute(config *Config) []error
	sftpGet(conf SftpConfig) error
	encryptFiles(config EncryptFilesConfig) (err error)
	sftpTo(conf SftpConfig) error
}

//Tasks Configuration
type Tasks struct {
	GetFilesFromBFP SftpConfig         `json:"getFilesFromBFP"`
	EncryptFiles    EncryptFilesConfig `json:"encrypteFiles"`
	SftpFilesToANZ  SftpConfig         `json:"sftpFilesToANZ"`
	SftpFilesToPx   SftpConfig         `json:"sftpFilesToPx"`
	SftpFilesToBNZ  SftpConfig         `json:"sftpFilesToBNZ"`
}

// Config defines the required arguements for the pipeline
type Config struct {
	Database DbConnection `json:"database"`
	Tasks    Tasks        `json:"tasks"`
}

// DbConnection stores connection information for the database
type DbConnection struct {
	// @todo pull from config
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Name     string `json:"name"`
	Timeout  string `json:"timeout"`
}

type pipeline struct {
	log           *log.Entry
	correlationID string
	DbConnection  DbConnection
	taskConfig    Config
}

// New Pipeline
func New(correlationID string) Pipeline {

	pipeline := &pipeline{
		log: log.WithFields(log.Fields{
			"correlationId": correlationID,
		}),
		correlationID: correlationID,
	}

	return pipeline
}

func (p pipeline) Execute(config *Config) (errorList []error) {

	p.log.Info("Starting Direct Debit Pipeline")

	// @todo config validation
	// @todo turn into loop
	bfpSftp := config.Tasks.GetFilesFromBFP
	if err := p.sftpTo(bfpSftp); err != nil {
		errorList = append(errorList, err)
	}

	encryptForANZ := config.Tasks.EncryptFiles
	if err := p.encryptFiles(encryptForANZ); err != nil {
		errorList = append(errorList, err)
	}

	anzSftp := config.Tasks.SftpFilesToANZ
	if err := p.sftpTo(anzSftp); err != nil {
		errorList = append(errorList, err)
	}

	pxSftp := config.Tasks.SftpFilesToPx
	if err := p.sftpTo(pxSftp); err != nil {
		errorList = append(errorList, err)
	}

	bnzSftp := config.Tasks.SftpFilesToBNZ
	if err := p.sftpTo(bnzSftp); err != nil {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		p.log.Warn("Finished DD Pipeline with Errors")
	} else {
		p.log.Info("Finished DD Pipeline Without Errors")
	}

	return errorList
}
