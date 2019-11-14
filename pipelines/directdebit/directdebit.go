package directdebit

import (
	"fmt"
	"strings"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

// Pipeline is an implementation of a pipeline
type Pipeline interface {
	Execute(correlationID string) (errorList []error)
	Close() error
	sftpGet(conf *SftpConfig) error
	sftpTo(conf *SftpConfig) error
	archiveTransferred(conf *ArchiveConfig) error
	cleanDirtyFiles(conf *CleanUpConfig) []error
	pgpEncryptFilesForBank(conf *EncryptFilesConfig) []error
}

//TasksConfig Configuration
type TasksConfig struct {
	GetFilesFromBFP    SftpConfig         `json:"getFilesFromBFP"`
	CleanBFP           SftpConfig         `json:"cleanBFP"`
	EncryptFiles       EncryptFilesConfig `json:"encrypteFiles"`
	SftpFilesToANZ     SftpConfig         `json:"sftpFilesToANZ"`
	SftpFilesToPx      SftpConfig         `json:"sftpFilesToPx"`
	SftpFilesToBNZ     SftpConfig         `json:"sftpFilesToBNZ"`
	ArchiveTransferred ArchiveConfig      `json:"archiveTransferred"`
	CleanDirtyFiles    CleanUpConfig      `json:"cleanDirtyFiles`
}

// Config defines the required arguements for the pipeline
type Config struct {
	Database mysql.Config `json:"database"`
	Tasks    TasksConfig  `json:"tasks"`
}

type ddPipeline struct {
	log           *log.Entry
	correlationID string
	transferlog   *TransferLog
	taskConfig    *Config
}

// New Pipeline
func New(config *Config, log *log.Entry) (Pipeline, error) {

	var p *ddPipeline

	if config.Database.Addr != "" {
		dbConfig := config.Database
		dbConfig.ParseTime = true

		redact := func(r rune) rune {
			return '*'
		}

		redactedPw := strings.Map(redact, dbConfig.Passwd)

		log.Debugf("Connection String (pw redacted): %s:%s@/%s", dbConfig.User, redactedPw, dbConfig.Addr)

		if err := mysql.SetLogger(log); err != nil {
			return nil, err
		}

		// if config.Database {
		connectionString := dbConfig.FormatDSN()
		db, err := gorm.Open("mysql", connectionString)
		if err != nil {
			return nil, fmt.Errorf("Unable to connect to the database: %s", err.Error())
		}
		db.SetLogger(log)
		db.LogMode(true)

		p = &ddPipeline{
			taskConfig:  config,
			log:         log,
			transferlog: NewRecorder(db, log),
		}
	} else {
		p = &ddPipeline{
			taskConfig: config,
			log:        log,
		}
	}

	return p, nil
}

// Execute starts the execution of the pipeline
func (p ddPipeline) Execute(correlationID string) (errorList []error) {
	p.correlationID = correlationID

	// @todo put this into a workflow
	p.log.Info("Starting Direct Debit Pipeline")

	// @todo config validation
	// @todo turn into loop
	if err := p.getFilesFromBFP(); err != nil {
		// we need the files from the BFP otherwise there is no point
		return append(errorList, err)
	}

	if err := p.cleanBFP(); err != nil {
		// not a big deal if cleaning fails..we can clean it up after
		errorList = append(errorList, err)
	}

	if err := p.encryptFiles(); err != nil {
		// We need all the files encrypted
		// before we continue further
		return err
	}

	// remove all the plain text files
	if err := p.cleanUp(); err != nil {
		errorList = append(errorList, err...)
	}

	// Transfer the files
	if err := p.sftpFilesToANZ(); err != nil {
		errorList = append(errorList, err)
	}

	if err := p.sftpFilesToPx(); err != nil {
		errorList = append(errorList, err)
	}

	if err := p.sftpFilesToBNZ(); err != nil {
		errorList = append(errorList, err)
	}

	// Archive the folder
	if err := p.archive(); err != nil {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		p.log.Error("END DD Pipeline with Errors")
	} else {
		p.log.Info("END DD Pipeline Without Errors")
	}

	return errorList
}

func (p ddPipeline) Close() error {
	if err := p.transferlog.Conn.Close(); err != nil {
		p.log.Warningf("Error closing database connecton, %s", err.Error())
	}
	return nil
}

func (p ddPipeline) archive() error {
	p.log.Info("Archiving Transferred Files")

	archiveConfig := p.taskConfig.Tasks.ArchiveTransferred
	if err := p.archiveTransferred(&archiveConfig); err != nil {
		p.log.Error(err.Error())
		return err
	}
	p.log.Info("Archiving Transferred Files Complete")
	return nil
}

func (p ddPipeline) cleanUp() []error {
	p.log.Info("Clean Up Start")
	cleanUpConfig := p.taskConfig.Tasks.CleanDirtyFiles
	err := p.cleanDirtyFiles(&cleanUpConfig)
	p.log.Info("Clean Up Complete")
	return err
}

func (p ddPipeline) getFilesFromBFP() error {

	p.log.Info("GetFilesFromBFP Start")
	bfpSftp := p.taskConfig.Tasks.GetFilesFromBFP
	if bfpSftp.Enabled {
		if err := p.sftpGet(&bfpSftp); err != nil {
			p.log.Error("Error Collecting the files. Unable to continue without files..Aborting")
			return err
		}
		p.log.Info("GetFilesFromBFP Complete")
		return nil
	}
	p.log.Warn("GetFilesFromBFP Skipped")

	return nil
}

func (p ddPipeline) cleanBFP() error {

	p.log.Info("CleanBFP Start")
	bfpClean := p.taskConfig.Tasks.CleanBFP
	if bfpClean.Enabled {
		if err := p.sftpClean(&bfpClean); err != nil {
			p.log.Warningf("Unable to clean remote dir %s", err.Error())
			return err
		}
		return nil
	}
	p.log.Warn("CleanBFP Skipped")
	return nil
}

func (p ddPipeline) encryptFiles() []error {
	p.log.Info("EncryptFiles Start")
	encryptionConfig := p.taskConfig.Tasks.EncryptFiles
	if encryptionConfig.Enabled {
		if err := p.pgpEncryptFilesForBank(&encryptionConfig); err != nil {
			p.log.Error("Unable to encrypt all files..Aborting")
			return err
		}
		p.log.Info("Encrypt Files Complete")
		return nil
	}
	p.log.Warn("Encrypt Files Skipped")
	return nil
}

func (p ddPipeline) sftpFilesToANZ() error {

	p.log.Info("SftpFilesToANZ Start")

	anzSftp := p.taskConfig.Tasks.SftpFilesToANZ
	if anzSftp.Enabled {
		if err := p.sftpTo(&anzSftp); err != nil {
			return err
		}
		p.log.Info("SftpFilesToANZ Complete")
		return nil
	}
	p.log.Warn("SftpFilesToANZ Skipped")

	return nil
}

func (p ddPipeline) sftpFilesToPx() error {
	p.log.Info("SftpFilesToPx Start")
	pxSftp := p.taskConfig.Tasks.SftpFilesToPx
	if pxSftp.Enabled {
		if err := p.sftpTo(&pxSftp); err != nil {
			return err
		}
		p.log.Info("SftpFilesToPx Complete")
		return nil
	}
	p.log.Warn("SftpFilesToPx Skipped")

	return nil
}

func (p ddPipeline) sftpFilesToBNZ() error {
	p.log.Info("SftpFilesToBNZ Start")

	bnzSftp := p.taskConfig.Tasks.SftpFilesToBNZ
	if bnzSftp.Enabled {
		if err := p.sftpTo(&bnzSftp); err != nil {
			return err
		}
		p.log.Info("SftpFilesToBNZ Complete")
		return nil
	}

	p.log.Warn("SftpFilesToBNZ Skipped")
	return nil
}
