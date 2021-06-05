package directdebit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Pipeline is an implementation of a pipeline
type Pipeline interface {
	StartListener(listenerError chan error)
	Execute(string) []error
	Close() error
	sftpGet(conf *SftpConfig) error                          // @todo this shouldn't be part of the generic interface
	sftpTo(conf *SftpConfig) error                           // @todo this shouldn't be part of the generic interface
	archiveTransferred(conf *ArchiveConfig) error            // @todo this shouldn't be part of the generic interface
	cleanDirtyFiles(conf *CleanUpConfig) []error             // @todo this shouldn't be part of the generic interface
	pgpEncryptFilesForBank(conf *EncryptFilesConfig) []error // @todo this shouldn't be part of the generic interface
}

//TasksConfig Configuration
type TasksConfig struct {
	GetFilesFromBFP    *SftpConfig         `json:"getFilesFromBFP"`
	CleanBFP           *SftpConfig         `json:"cleanBFP"`
	EncryptFiles       *EncryptFilesConfig `json:"encryptFiles"`
	SftpFilesToANZ     *SftpConfig         `json:"sftpFilesToANZ"`
	SftpFilesToPx      *SftpConfig         `json:"sftpFilesToPx"`
	SftpFilesToBNZ     *SftpConfig         `json:"sftpFilesToBNZ"`
	ArchiveTransferred *ArchiveConfig      `json:"archiveTransferred"`
	CleanDirtyFiles    *CleanUpConfig
}

// PipelineConfig defines the required arguements for the pipeline
type PipelineConfig struct {
	Database mysql.Config
	Rabbitmq *BusConfig
	Tasks    *TasksConfig
}

type ddPipeline struct {
	log           *log.Entry
	correlationID string
	consumer      *MessageConsumer
	transferlog   *TransferLog
	encryptionLog *EncryptionLog
	taskConfig    *PipelineConfig
}

// New Pipeline
func New(c *PipelineConfig) (Pipeline, error) {

	log := log.WithField("Pipeline", "DirectDebit")

	var p *ddPipeline = &ddPipeline{
		taskConfig: c,
		log:        log,
	}

	if c.Database.Addr != "" {
		db, err := connectToDb(c.Database)
		if err != nil {
			return nil, err
		}
		db.SetLogger(p.log)
		db.LogMode(true)
		p.transferlog = NewTransferRecorder(db, p.log)
		p.encryptionLog = NewEncryptionRecorder(db, p.log)
	}

	if c.Rabbitmq.Host != "" {
		p.consumer = NewConsumer(c.Rabbitmq, p.log)
	}

	return p, nil
}

func connectToDb(dbConfig mysql.Config) (*gorm.DB, error) {

	dbConfig.ParseTime = true

	redact := func(r rune) rune {
		return '*'
	}

	redactedPw := strings.Map(redact, dbConfig.Passwd)

	log.Debugf("Connection String (pw redacted): %s:%s@/%s", dbConfig.User, redactedPw, dbConfig.Addr)

	// if err := mysql.SetLogger(p.log); err != nil {
	// 	return nil, err
	// }

	// if config.Database {
	connectionString := dbConfig.FormatDSN()
	db, err := gorm.Open("mysql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to the database: %s", err.Error())
	}
	return db, err
}

func (p *ddPipeline) StartListener(listenerError chan error) {

	conn, err := p.consumer.Connect()
	if err != nil {
		listenerError <- err
		// goroutine will block forever if we don't return
		return
	}

	if conn == nil || conn.IsClosed() {
		listenerError <- fmt.Errorf("RabbitMQ Connection is in an unexpected state")
		// goroutine will block forever if we don't return
		return
	}

	// we want to know if the connection get's closed
	rabbitCloseError := make(chan *amqp.Error)
	conn.NotifyClose(rabbitCloseError)

	p.log.Debug("Creating Channel")
	consumerCh, err := conn.Channel()
	if err != nil {
		p.log.Errorf("Unable to create Channel : %s ", err.Error())
		listenerError <- err
		close(rabbitCloseError)

		// goroutine will block forever if we don't return
		return
	}

	p.log.Debug("Creating Exchanges and Queues")
	// Setup the Exchanges and the Queues
	if err := p.consumer.Configure(consumerCh); err != nil {
		listenerError <- err
		return
	}

	p.log.Info("Opening Consumer Channel")
	firehose, err := consumerCh.Consume(
		p.consumer.config.Queues[0].Name,
		"pipefire",
		false,
		false,
		false,
		false,
		nil)

	if err != nil {
		// channelError <- err
		// goroutine will block forever if we don't return
		listenerError <- err
		return
	}

	for {
		select {
		case err := <-rabbitCloseError:
			if conn != nil && !conn.IsClosed() {
				// can't imagine how it wwuld get to here
				// but handle it to be safe
				_ = conn.Close()
			}
			_ = consumerCh.Cancel("pipefire", false)
			p.log.Warning("RabbitMQ Connection has gone away")
			listenerError <- err

			p.log.Info("Shutting Down Listener")
			return
		case msg := <-firehose:

			if msg.Body == nil || len(msg.Body) < 2 {
				break
			}

			p.log.Debugf("Message [%s] Correlation ID: %s ", msg.Body, msg.CorrelationId)
			payload := &TransferFilesPayload{}

			err := json.Unmarshal(msg.Body, payload)
			if err != nil {
				// @todo move to error queue
				p.log.Errorf("Unable to unmarshall payload")
				msg.Reject(false)
				break
			}

			// de-serialise
			if payload != nil && payload.Message.CorrelationID == "00000000-0000-0000-0000-000000000000" {
				payload.Message.CorrelationID = uuid.New().String()
				// this is useless so make a random one and log it
				p.log.Warnf("CorrelationID has not been set correctly, setting to a random GUID %s :", payload.Message.CorrelationID)
				// @todo move to error queue
			}

			errList := p.Execute(payload.Message.CorrelationID)
			if len(errList) > 0 {
				p.log.Info("Direct Debit Run Finished With Errors")
				for _, e := range errList {
					p.log.Errorf("%s ", e.Error())
				}
				// don't requeue at this stage
				msg.Nack(false, false)
			} else {
				p.log.Info("Direct Debit Run Completed Successfully")
				msg.Ack(true)
			}

		}
	}
}

// Execute starts the execution of the pipeline
func (p *ddPipeline) Execute(correlationID string) (errorList []error) {

	p.correlationID = correlationID
	p.log = log.WithField("correlationId", correlationID)

	// @todo put this into a workflow
	log.Info("Starting Direct Debit Pipeline")

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

	// Transfer the files
	if err := p.sftpFilesToANZ(); err != nil {
		errorList = append(errorList, err)
	}

	if err := p.sftpFilesToPx(); err != nil {
		errorList = append(errorList, err)
	}

	// Archive the folder
	if err := p.archive(); err != nil {
		errorList = append(errorList, err)
	}

	// remove all the plain text files
	if err := p.cleanUp(); err != nil {
		errorList = append(errorList, err...)
	}

	if len(errorList) > 0 {
		log.Error("END DD Pipeline with Errors")
	} else {
		log.Info("END DD Pipeline Without Errors")
	}

	return errorList
}

func (p *ddPipeline) Close() error {
	p.log.Info("Recieved Shutdown Request")
	if p.transferlog != nil && p.transferlog.Conn != nil {
		p.log.Info("Shutdown Database Connection")
		if err := p.transferlog.Conn.Close(); err != nil {
			p.log.Warningf("Error closing database connecton, %s", err.Error())
		}
		p.log.Info("Shutdown Database Complete")
	}

	if p.consumer != nil {
		p.log.Info("Shutdown RabbitMQ Connection")
		consumer := *p.consumer
		if err := consumer.Close(); err != nil {
			p.log.Warningf("Error closing RabbitMQ connecton, %s", err.Error())
			return err
		}
		p.log.Info("Shutdown RabbitMQ Complete")
	}

	p.log.Info("Shutdown Complete")
	return nil
}

func (p *ddPipeline) archive() error {
	p.log.Info("Archiving Transferred Files")

	archiveConfig := p.taskConfig.Tasks.ArchiveTransferred
	if archiveConfig.Enabled {
		if err := p.archiveTransferred(archiveConfig); err != nil {
			p.log.Error(err.Error())
			return err
		}
		p.log.Info("Archiving Transferred Files Complete")
	} else {
		p.log.Warn("Archiving Transferred Files Skipped")
	}

	return nil
}

func (p *ddPipeline) cleanUp() (err []error) {
	p.log.Info("Clean Up Start")
	cleanUpConfig := p.taskConfig.Tasks.CleanDirtyFiles
	if cleanUpConfig.Enabled {
		err = p.cleanDirtyFiles(cleanUpConfig)
		p.log.Info("Clean Up Complete")
	} else {
		p.log.Warn("Clean Up Files Skipped")
	}

	return err
}

func (p *ddPipeline) getFilesFromBFP() error {

	p.log.Info("GetFilesFromBFP Start")
	bfpSftp := p.taskConfig.Tasks.GetFilesFromBFP
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

func (p *ddPipeline) cleanBFP() error {

	p.log.Info("CleanBFP Start")
	bfpClean := p.taskConfig.Tasks.CleanBFP
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

func (p *ddPipeline) encryptFiles() []error {
	p.log.Info("EncryptFiles Start")
	encryptionConfig := p.taskConfig.Tasks.EncryptFiles
	if encryptionConfig != nil && encryptionConfig.Enabled {
		if err := p.pgpEncryptFilesForBank(encryptionConfig); err != nil {
			p.log.Error("Unable to encrypt all files..Aborting")
			return err
		}
		p.log.Info("Encrypt Files Complete")
		return nil
	}
	p.log.Warn("Encrypt Files Skipped")
	return nil
}

func (p *ddPipeline) sftpFilesToANZ() error {

	p.log.Info("SftpFilesToANZ Start")

	anzSftp := p.taskConfig.Tasks.SftpFilesToANZ
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

func (p *ddPipeline) sftpFilesToPx() error {
	p.log.Info("SftpFilesToPx Start")
	pxSftp := p.taskConfig.Tasks.SftpFilesToPx
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
