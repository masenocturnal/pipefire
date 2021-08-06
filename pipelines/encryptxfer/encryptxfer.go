package encryptxfer

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/masenocturnal/pipefire/internal/config"
	"github.com/masenocturnal/pipefire/internal/crypto"
	"github.com/masenocturnal/pipefire/internal/db"
	"github.com/masenocturnal/pipefire/internal/encryption_recorder"

	"github.com/masenocturnal/pipefire/internal/mq"
	"github.com/masenocturnal/pipefire/internal/transfer_recorder"
	"github.com/masenocturnal/pipefire/tasks/archive"
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
	Task          string `json:"task"`
	StartDate     string `json:"start_date"`
	CorrelationID string `json:"correlationId"`
}

//ArchiveConfig configuration for the archive task
type ArchiveConfig struct {
	Src     string
	Dest    string
	Enabled bool
}

//TasksConfig Configuration
type TasksConfig struct {
	EncryptFiles       *crypto.EncryptFilesConfig `json:"encryptFiles"`
	ArchiveTransferred *archive.ArchiveConfig     `json:"archiveTransferred"`
}

type encryptxferPipeline struct {
	log           *log.Entry
	correlationID string
	taskConfig    *config.PipelineConfig
	transferlog   *transfer_recorder.TransferLog
	encryptionLog *encryption_recorder.EncryptionLog
	consumer      *mq.MessageConsumer
}

// New Pipeline
func New(c *config.PipelineConfig) (*encryptxferPipeline, error) {

	log := log.WithField("Pipeline", "DirectDebit")

	var p *encryptxferPipeline = &encryptxferPipeline{
		taskConfig: c,
		log:        log,
	}

	if c.Database.Addr != "" {
		db, err := db.ConnectToDb(c.Database)
		if err != nil {
			return nil, err
		}
		db.SetLogger(p.log)
		db.LogMode(true)
		p.transferlog = transfer_recorder.NewTransferRecorder(db, p.log)
		p.encryptionLog = encryption_recorder.NewEncryptionRecorder(db, p.log)
	}

	if c.Rabbitmq.Host != "" {
		p.consumer = mq.NewConsumer(c.Rabbitmq, p.log)
	}

	return p, nil
}

func (p *encryptxferPipeline) StartListener(listenerError chan error) {

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
		p.consumer.Config.Queues[0].Name,
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

func (p *encryptxferPipeline) GetCorrelationId() string {
	return p.correlationID
}

// Execute starts the execution of the pipeline
func (p *encryptxferPipeline) Execute(correlationID string) (errorList []error) {

	p.correlationID = correlationID
	p.log = log.WithField("correlationId", correlationID)

	if len(errorList) > 0 {
		log.Error("END DD Pipeline with Errors")
	} else {
		log.Info("END DD Pipeline Without Errors")
	}

	return errorList
}

func (p *encryptxferPipeline) Close() error {
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

func (p *encryptxferPipeline) Getlogger() *logrus.Logger {
	return p.log.Logger
}
