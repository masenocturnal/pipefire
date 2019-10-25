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
	if err := p.getFilesFromBFP(config); err != nil {
		// we need the files from the BFP otherwise there is no point
		return append(errorList, err)
	}

	if err := p.cleanBFP(config); err != nil {
		// not a big deal if cleaning fails..we can clean it up after
		errorList = append(errorList, err)
	}

	if err := p.encrypteFiles(config); err != nil {
		// We need all the files encrypted
		// before we continue further
		return err
	}

	if err := p.sftpFilesToANZ(config); err != nil {
		errorList = append(errorList, err)
	}

	if err := p.sftpFilesToPx(config); err != nil {
		errorList = append(errorList, err)
	}

	if err := p.sftpFilesToBNZ(config); err != nil {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		p.log.Error("Finished DD Pipeline with Errors")
	} else {
		p.log.Info("Finished DD Pipeline Without Errors")
	}

	return errorList
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
