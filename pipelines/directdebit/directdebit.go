package directdebit

import (
	log "github.com/sirupsen/logrus"
)

// Pipeline is an implementation of a pipeline
type Pipeline interface {
	Execute(config *Config) []error
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
func New(correlationID string, log *log.Entry) Pipeline {

	pipeline := &pipeline{
		log:           log,
		correlationID: correlationID,
	}

	return pipeline
}

func (p pipeline) Execute(config *Config) (errorList []error) {

	p.log.Info("Starting Direct Debit Pipeline")

	// @todo config validation
	// @todo turn into loop
	p.log.Info("GetFilesFromBFP Start")
	bfpSftp := config.Tasks.GetFilesFromBFP
	if bfpSftp.Enabled {
		if err := p.sftpGet(bfpSftp); err != nil {
			errorList = append(errorList, err)
			p.log.Error("Error Collecting the files. Unable to continue without files..Aborting")
			return errorList
		}
		p.log.Info("GetFilesFromBFP Complete")
	} else {
		p.log.Info("GetFilesFromBFP Skipped")
	}

	p.log.Info("CleanBFP Start")
	bfpClean := config.Tasks.CleanBFP
	if bfpClean.Enabled {
		if err := p.sftpClean(bfpClean); err != nil {
			errorList = append(errorList, err)
			p.log.Warningf("Unable to clean remote dir %s", err.Error())
		}
	} else {
		p.log.Info("CleanBFP Skipped")
	}

	p.log.Info("EncryptFiles Start")
	encryptionConfig := config.Tasks.EncryptFiles
	if encryptionConfig.Enabled {
		if err := p.encryptFiles(encryptionConfig); err != nil {

			for _, e := range err {
				errorList = append(errorList, e)
			}
			p.log.Error("Unable to encrypt all files..Aborting")
			return errorList
		}
		p.log.Info("SftpFilesToANZ Complete")
	} else {
		p.log.Info("SftpFilesToANZ Skipped")
	}

	p.log.Info("SftpFilesToANZ Start")
	anzSftp := config.Tasks.SftpFilesToANZ
	if anzSftp.Enabled {
		if err := p.sftpTo(anzSftp); err != nil {
			errorList = append(errorList, err)
		}
		p.log.Info("SftpFilesToANZ Complete")
	} else {
		p.log.Info("SftpFilesToANZ Skipped")
	}

	p.log.Info("SftpFilesToPx Start")
	pxSftp := config.Tasks.SftpFilesToPx
	if pxSftp.Enabled {
		if err := p.sftpTo(pxSftp); err != nil {
			errorList = append(errorList, err)
		}
		p.log.Info("SftpFilesToPx Complete")
	} else {
		p.log.Info("SftpFilesToPx Skipped")
	}

	p.log.Info("SftpFilesToBNZ Start")
	bnzSftp := config.Tasks.SftpFilesToBNZ
	if bnzSftp.Enabled {
		if err := p.sftpTo(bnzSftp); err != nil {
			errorList = append(errorList, err)
		}
		p.log.Info("SftpFilesToBNZ Complete")
	} else {
		p.log.Info("SftpFilesToBNZ Skipped")
	}

	if len(errorList) > 0 {
		p.log.Warn("Finished DD Pipeline with Errors")
	} else {
		p.log.Info("Finished DD Pipeline Without Errors")
	}

	return errorList
}
