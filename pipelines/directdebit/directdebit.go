package main

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/masenocturnal/pipefire/internal/config"
	"github.com/masenocturnal/pipefire/internal/db"
	"github.com/masenocturnal/pipefire/internal/encryption_recorder"
	"github.com/masenocturnal/pipefire/internal/mq"
	"github.com/masenocturnal/pipefire/internal/transfer_recorder"

	"github.com/masenocturnal/pipefire/internal/sftp"
	"github.com/masenocturnal/pipefire/tasks/archive"
	"github.com/masenocturnal/pipefire/tasks/cleanup"
	"github.com/masenocturnal/pipefire/tasks/encryption"
	sftpTask "github.com/masenocturnal/pipefire/tasks/sftp"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

//TransferFilesPayload represents the payload received from the message bus
type TransferFilesPayload struct {
	MessageType []string
	Message     MessagePayload
}

//MessagePayload represents the message content in a TransferFilesPayload from the message bus
type MessagePayload struct {
	Task          string               `json:"task"`
	StartDate     string               `json:"start_date"`
	CorrelationID string               `json:"correlationId"`
	Files         []sftp.TransferFiles `json:"files,omitempty"`
}

type CustomPipeline struct {
	Log            *log.Entry
	correlationID  string
	consumer       *mq.MessageConsumer
	pipelineConfig *config.PipelineConfig
}

const version string = "0.0.1"

var transferLog *transfer_recorder.TransferLog
var encryptionLog *encryption_recorder.EncryptionLog
var consumer *mq.MessageConsumer

func GetVersion() string {
	return version
}

// New Pipeline
func New(c *config.PipelineConfig) (interface{}, error) {

	log := log.WithField("Pipeline", "DirectDebit")

	p := &CustomPipeline{
		pipelineConfig: c,
		Log:            log,
	}

	if c.Database.Addr != "" {
		db, err := db.ConnectToDb(c.Database)
		if err != nil {
			return nil, err
		}
		db.SetLogger(p.Log)
		db.LogMode(true)
		transferLog = transfer_recorder.NewTransferRecorder(db, p.Log)
		encryptionLog = encryption_recorder.NewEncryptionRecorder(db, p.Log)
	}

	if c.Rabbitmq.Host != "" {
		consumer = mq.NewConsumer(c.Rabbitmq, p.Log)
	}

	return p, nil
}

func (p *CustomPipeline) StartListener(listenerError chan error) {

	conn, err := consumer.Connect()
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

	p.Log.Debug("Creating Channel")
	consumerCh, err := conn.Channel()
	if err != nil {
		p.Log.Errorf("Unable to create Channel : %s ", err.Error())
		listenerError <- err
		close(rabbitCloseError)

		// goroutine will block forever if we don't return
		return
	}

	p.Log.Debug("Creating Exchanges and Queues")
	// Setup the Exchanges and the Queues
	if err := consumer.Configure(consumerCh); err != nil {
		listenerError <- err
		return
	}

	p.Log.Info("Opening Consumer Channel")
	firehose, err := consumerCh.Consume(
		consumer.Config.Queues[0].Name,
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
			p.Log.Warning("RabbitMQ Connection has gone away")
			listenerError <- err

			p.Log.Info("Shutting Down Listener")
			return
		case msg := <-firehose:

			if msg.Body == nil || len(msg.Body) < 2 {
				break
			}

			p.Log.Debugf("Message [%s] Correlation ID: %s ", msg.Body, msg.CorrelationId)
			payload := &TransferFilesPayload{}

			err := json.Unmarshal(msg.Body, payload)
			if err != nil {
				// @todo move to error queue
				p.Log.Errorf("Unable to unmarshall payload")
				msg.Reject(false)
				break
			}

			// de-serialise
			if payload != nil && payload.Message.CorrelationID == "00000000-0000-0000-0000-000000000000" {
				payload.Message.CorrelationID = uuid.New().String()
				// this is useless so make a random one and log it
				p.Log.Warnf("CorrelationID has not been set correctly, setting to a random GUID %s :", payload.Message.CorrelationID)
				// @todo move to error queue
			}

			errList := p.Execute(payload.Message)
			if len(errList) > 0 {
				p.Log.Info("Direct Debit Run Finished With Errors")
				for _, e := range errList {
					p.Log.Errorf("%s ", e.Error())
				}
				// don't requeue at this stage
				msg.Nack(false, false)
			} else {
				p.Log.Info("Direct Debit Run Completed Successfully")
				msg.Ack(true)
			}

		}
	}
}

func (p *CustomPipeline) GetCorrelationID() string {
	return p.correlationID
}
func (p *CustomPipeline) SetCorrelationID(correlationID string) {
	p.correlationID = correlationID
}

// Execute starts the execution of the pipeline
func (p *CustomPipeline) Execute(msg interface{}) (errorList []error) {

	payload := msg.(MessagePayload)
	p.correlationID = payload.CorrelationID
	p.Log = log.WithField("correlationId", p.correlationID)

	// @todo put this into a workflow
	log.Info("Starting Direct Debit Pipeline")

	// this needs to be dynamic based on what's there
	// @todo config validation
	// @todo turn into loop
	for _, task := range p.pipelineConfig.Tasks {

		task.TaskConfiguration = string(task.TaskConfig)

		filesToXfer := payload.Files

		if task.Enabled {

			p.Log.Info("Start: " + task.Name)

			switch task.Type {
			case "sftp.get":

				sftpConfig, err := sftpTask.GetConfig(task.TaskConfiguration)
				if err != nil {
					return append(errorList, err)
				}

				if err := sftpTask.SFTPGet(sftpConfig, &filesToXfer, p.Log); err != nil {
					p.Log.Error("Error Collecting the files. Unable to continue without files..Aborting")
					return append(errorList, err)
				}

			case "sftp.clean":
				sftpConfig, err := sftpTask.GetConfig(task.TaskConfiguration)
				if err != nil {
					return append(errorList, err)
				}

				if err := sftpTask.SFTPClean(sftpConfig, &filesToXfer, p.Log); err != nil {
					p.Log.Warningf("Unable to clean remote dir %s", err.Error())
					return append(errorList, err)
				}

			case "encrypt":
				if err := p.encryptFiles(task); err != nil {
					// We need all the files encrypted
					// before we continue further
					return err
				}
			case "sftp.put":
				// Transfer the files
				if err := p.sftpPut(task); err != nil {
					errorList = append(errorList, err)
				}
			case "archive":
				// Archive the folder
				if err := p.archive(task); err != nil {
					errorList = append(errorList, err)
				}
			case "cleanup":
				// remove all the plain text files
				if err := p.cleanUp(task); err != nil {
					errorList = append(errorList, err...)
				}
			}
			p.Log.Info("Tasl " + task.Name + " Complete")
		} else {
			p.Log.Warn("Task " + task.Name + " Skipped - disabled in config")
		}

	}

	if len(errorList) > 0 {
		log.Error("END DD Pipeline with Errors")
	} else {
		log.Info("END DD Pipeline Without Errors")
	}

	return errorList
}

func (p *CustomPipeline) Close() error {
	p.Log.Info("Recieved Shutdown Request")
	if transferLog != nil && transferLog.Conn != nil {
		p.Log.Info("Shutdown Database Connection")
		if err := transferLog.Conn.Close(); err != nil {
			p.Log.Warningf("Error closing database connecton, %s", err.Error())
		}
		p.Log.Info("Shutdown Database Complete")
	}

	if consumer != nil {
		p.Log.Info("Shutdown RabbitMQ Connection")
		consumer := *consumer
		if err := consumer.Close(); err != nil {
			p.Log.Warningf("Error closing RabbitMQ connecton, %s", err.Error())
			return err
		}
		p.Log.Info("Shutdown RabbitMQ Complete")
	}

	p.Log.Info("Shutdown Complete")
	return nil
}

func (p *CustomPipeline) archive(taskDefinition *config.TaskDefinition) error {

	archiveConfig, err := archive.GetConfig(taskDefinition.TaskConfiguration)
	if err != nil {
		return err
	}

	if err := archive.ArchiveTransferred(archiveConfig, *p.Log); err != nil {
		p.Log.Error(err.Error())
		return err
	}

	return nil
}

func (p *CustomPipeline) cleanUp(taskDefinition *config.TaskDefinition) (errs []error) {

	cleanUpConfig, err := cleanup.GetConfig(taskDefinition.TaskConfiguration)
	if err != nil {
		return append(errs, err)
	}

	errs = cleanup.CleanDirtyFiles(cleanUpConfig, p.Log)
	if errs != nil {
		return errs
	}

	return errs
}

func (p *CustomPipeline) encryptFiles(taskDefinition *config.TaskDefinition) (errs []error) {

	config, err := encryption.GetConfig(taskDefinition.TaskConfiguration)
	if err != nil {
		return append(errs, err)
	}

	if err := encryption.PGPEncryptFilesForTransfer(config, encryptionLog, p.correlationID, p.Log); err != nil {
		p.Log.Error("Unable to encrypt all files..Aborting")
		return err
	}

	return nil

}

func (p *CustomPipeline) GetLogger() *logrus.Entry {
	return p.Log
}

func (p *CustomPipeline) SetLogger(l *logrus.Entry) {
	p.Log = l
}

func (p *CustomPipeline) sftpPut(taskDefinition *config.TaskDefinition) error {

	sftpConfig, err := sftpTask.GetConfig(taskDefinition.TaskConfiguration)
	if err != nil {
		return err
	}

	if err := sftpTask.SFTPTo(sftpConfig, transferLog, p.correlationID, p.Log); err != nil {
		return err
	}

	return nil
}

// func (p *CustomPipeline) sftpFilesToPx(taskDefinition *config.TaskDefinition) error {
// 	p.Log.Info("SftpFilesToPx Start")

// 	if pxSftp.Enabled {
// 		if err := sftpTask.SFTPTo(taskConfig, p.transferlog, p.correlationID, p.Log); err != nil {
// 			return err
// 		}
// 		p.Log.Info("SftpFilesToPx Complete")
// 		return nil
// 	}
// 	p.Log.Warn("SftpFilesToPx Skipped")

// 	return nil
// }
