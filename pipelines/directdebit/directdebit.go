package directdebit

import (
	"database/sql"
	"fmt"
	"strings"

	xferlog "github.com/masenocturnal/pipefire/pipelines/directdebit/lib"
	log "github.com/sirupsen/logrus"
)

// Pipeline is an implementation of a pipeline
type Pipeline interface {
	Execute() (errorList []error)
	Close() error
	sftpGet(conf SftpConfig) error
	encryptFiles(config EncryptFilesConfig) (err []error)
	sftpTo(conf SftpConfig) error
}

//Tasks Configuration
type Tasks struct {
	GetFilesFromBFP SftpConfig         `json:"getFilesFromBFP"`
	CleanBFP        SftpConfig         `json:"cleanBFP"`
	EncryptFiles    EncryptFilesConfig `json:"encrypteFiles"`
	SftpFilesToANZ  SftpConfig         `json:"sftpFilesToANZ"`
	SftpFilesToPx   SftpConfig         `json:"sftpFilesToPx"`
	SftpFilesToBNZ  SftpConfig         `json:"sftpFilesToBNZ"`
}

// Config defines the required arguements for the pipeline
type Config struct {
	Database xferlog.DbConfig `json:"database"`
	Tasks    Tasks            `json:"tasks"`
}

type pipeline struct {
	log           *log.Entry
	correlationID string
	transferlog   xferlog.TransferLog
	taskConfig    *Config
}

// New Pipeline
func New(config *Config, log *log.Entry) (Pipeline, error) {

	dbConfig := config.Database

	redact := func(r rune) rune {
		return '*'
	}

	redactedPw := strings.Map(redact, dbConfig.Password)

	log.Debugf("Connection String (pw redacted): %s:%s@/%s", dbConfig.Username, redactedPw, dbConfig.Host)

	connectionString := fmt.Sprintf("%s:%s@/%s", dbConfig.Username, dbConfig.Password, dbConfig.Host)
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	pipeline := &pipeline{
		taskConfig:  config,
		log:         log,
		transferlog: xferlog.NewTransferLog(db, log),
	}

	return pipeline, err
}

// Execute starts the execution of the pipeline
func (p pipeline) Execute() (errorList []error) {

	p.log.Info("Starting Direct Debit Pipeline")

	// @todo config validation
	// @todo turn into loop
	if err := p.getFilesFromBFP(p.taskConfig); err != nil {
		// we need the files from the BFP otherwise there is no point
		return append(errorList, err)
	}

	if err := p.cleanBFP(p.taskConfig); err != nil {
		// not a big deal if cleaning fails..we can clean it up after
		errorList = append(errorList, err)
	}

	if err := p.encrypteFiles(p.taskConfig); err != nil {
		// We need all the files encrypted
		// before we continue further
		return err
	}

	if err := p.sftpFilesToANZ(p.taskConfig); err != nil {
		errorList = append(errorList, err)
	}

	if err := p.sftpFilesToPx(p.taskConfig); err != nil {
		errorList = append(errorList, err)
	}

	if err := p.sftpFilesToBNZ(p.taskConfig); err != nil {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		p.log.Error("Finished DD Pipeline with Errors")
	} else {
		p.log.Info("Finished DD Pipeline Without Errors")
	}

	return errorList
}

func (p pipeline) Close() error {
	return p.transferlog.Close()
}

func (p pipeline) getFilesFromBFP(config *Config) error {

	p.log.Info("GetFilesFromBFP Start")
	bfpSftp := config.Tasks.GetFilesFromBFP
	if bfpSftp.Enabled {
		if err := p.sftpGet(bfpSftp); err != nil {
			p.log.Error("Error Collecting the files. Unable to continue without files..Aborting")
			return err
		}
		p.log.Info("GetFilesFromBFP Complete")
		return nil
	}
	p.log.Warn("GetFilesFromBFP Skipped")

	return nil
}

func (p pipeline) cleanBFP(config *Config) error {

	p.log.Info("CleanBFP Start")
	bfpClean := config.Tasks.CleanBFP
	if bfpClean.Enabled {
		if err := p.sftpClean(bfpClean); err != nil {
			p.log.Warningf("Unable to clean remote dir %s", err.Error())
			return err
		}
		return nil
	}
	p.log.Warn("CleanBFP Skipped")
	return nil
}

func (p pipeline) encrypteFiles(config *Config) []error {
	p.log.Info("EncryptFiles Start")
	encryptionConfig := config.Tasks.EncryptFiles
	if encryptionConfig.Enabled {
		if err := p.encryptFiles(encryptionConfig); err != nil {
			p.log.Error("Unable to encrypt all files..Aborting")
			return err
		}
		p.log.Info("SftpFilesToANZ Complete")
		return nil
	}
	p.log.Warn("SftpFilesToANZ Skipped")
	return nil
}

func (p pipeline) sftpFilesToANZ(config *Config) error {
	p.log.Info("SftpFilesToANZ Start")
	anzSftp := config.Tasks.SftpFilesToANZ
	if anzSftp.Enabled {
		if err := p.sftpTo(anzSftp); err != nil {
			return err
		}
		p.log.Info("SftpFilesToANZ Complete")
		return nil
	}
	p.log.Warn("SftpFilesToANZ Skipped")

	return nil
}

func (p pipeline) sftpFilesToPx(config *Config) error {
	p.log.Info("SftpFilesToPx Start")
	pxSftp := config.Tasks.SftpFilesToPx
	if pxSftp.Enabled {
		if err := p.sftpTo(pxSftp); err != nil {
			return err
		}
		p.log.Info("SftpFilesToPx Complete")
		return nil
	}
	p.log.Warn("SftpFilesToPx Skipped")

	return nil
}

func (p pipeline) sftpFilesToBNZ(config *Config) error {
	p.log.Info("SftpFilesToBNZ Start")

	bnzSftp := config.Tasks.SftpFilesToBNZ
	if bnzSftp.Enabled {
		if err := p.sftpTo(bnzSftp); err != nil {
			return err
		}
		p.log.Info("SftpFilesToBNZ Complete")
		return nil
	}

	p.log.Warn("SftpFilesToBNZ Skipped")
	return nil
}
