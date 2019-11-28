package directdebit

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

//BusConfig RabbitMQ configuration definition
type BusConfig struct {
	User      string
	Password  string
	Host      string
	Port      string
	Vhost     string
	Exchanges []*ExchangeConfig
	Queues    []*QueueConfig `json:"queues"`
}

//ExchangeConfig RabbbitMQ Exchange configuration
type ExchangeConfig struct {
	Name         string
	ExchangeType string
	Durable      bool
}

//QueueConfig RabbitMQ Queue definition
type QueueConfig struct {
	Name           string
	Durable        bool            `json:"durable"`
	DeleteOnUnused bool            `json:"deleteOnUnused"`
	Exclusive      bool            `json:"exclusive"`
	NoWait         bool            `json:"noWait"`
	Args           string          `json:"args"`
	Bindings       []BindingConfig `json:"bindings"`
}

//BindingConfig Queue/Exchange Bindings
type BindingConfig struct {
	RoutingKey string
	Exchange   string
}

//QueueClient Consumes a message for the pipeline
type QueueClient interface {
	Configure() error
	Close() error
}

//MessageConsumer holds the configuration, current connection and open channel
type MessageConsumer struct {
	config          *BusConfig
	Connection      *amqp.Connection
	ConsumerChannel *amqp.Channel
	log             *log.Entry
	Shutdown        bool
}

//TransferFilesPayload represents the payload received from the message bus
type TransferFilesPayload struct {
	MessageType []string
	Message     MessagePayload
}

//MessagePayload represents the message content in a TransferFilesPayload from the message bus
type MessagePayload struct {
	Task          string
	StartDate     string `json:"start_date"`
	CorrelationID string
}

//NewConsumer provides an instance of MessageConsumer
func NewConsumer(config *BusConfig, log *log.Entry) *MessageConsumer {

	consumer := &MessageConsumer{
		log:    log.WithField("Component", "Consumer"),
		config: config,
	}
	return consumer
}

func (c *MessageConsumer) reconnect(status chan string) {

	go c.connect(status)
	if <-status == "connected"

}

func (c *MessageConsumer) connect(status chan string) {
	// var rabbitErr *amqp.Error
	rabbitCloseError := make(chan *amqp.Error)
	channelError := make(chan *amqp.Error)
	connectionError := make(chan bool)

	// don't try and reconnect if we have intentionally shutdown
	if !c.Shutdown && (c.Connection == nil || c.Connection.IsClosed()) {

		go func() {
			var err error
			uri := c.config.ConnectionString()
			c.log.Info("Try and connect")

			c.Connection, err = amqp.Dial(uri)

			if err != nil {
				c.log.Error(err.Error())
				// try again
				connectionError <- true

			} else {
				c.log.Info("Connected")
				c.Connection.NotifyClose(rabbitCloseError)

				c.log.Debug("Creating Channel")
				consumerCh, err := c.Connection.Channel()
				if err != nil {
					c.log.Errorf("Unable to create Channel : %s ", err.Error())
				}

				c.log.Debug("Configure Exchanges and Queues")
				if err := Configure(consumerCh, c.config); err != nil {
					c.log.Errorf("Unable to register Exchanges and Queues : %s ", err.Error())
				}

				c.log.Debug("Subscribe to close notifications")
				consumerCh.NotifyClose(channelError)
				c.ConsumerChannel = consumerCh

				c.log.Debug("Setup Complete")
				in <- true
			}
			// always return so we don't leak routines
			return
		}()

	}

	select {
	case op1 := <-rabbitCloseError:
		c.log.Warningf("Connection Closed %s", op1)
		closed := c.Connection.IsClosed()
		c.log.Infof("Connection Is %v", closed)
		time.Sleep(1 * time.Second)
		c.log.Infof("Restablishing Connection %s", op1)
		c.reconnect(in)

	case <-connectionError:
		c.log.Info("Connection Error")
		time.Sleep(1 * time.Second)
		c.reconnect(in)

	case <-channelError:
		c.log.Info("Channel went away but the connection is still up ?")
		c.reconnect(in)

	}
	// don't leak goroutines
	return
}

//Configure Creates the Exchange and Queue
func Configure(ch *amqp.Channel, config *BusConfig) (err error) {

	if len(config.Exchanges) > 0 {
		for _, ex := range config.Exchanges {
			err = ch.ExchangeDeclare(
				ex.Name,         // name
				ex.ExchangeType, // type
				ex.Durable,      // durable
				false,           // auto-deleted
				false,           // internal
				false,           // no-wait
				nil,             // arguments
			)
			if err != nil {
				return
			}
		}
	}

	if len(config.Queues) > 0 {
		for _, qConfig := range config.Queues {
			q, err := ch.QueueDeclare(
				qConfig.Name,           // name
				qConfig.Durable,        // durable
				qConfig.DeleteOnUnused, // delete when unused
				qConfig.Exclusive,      // exclusive
				qConfig.NoWait,         // no-wait
				nil,                    // arguments
			)
			if err != nil {
				return err
			}

			if len(qConfig.Bindings) > 0 {
				for _, bindingConfig := range qConfig.Bindings {
					err := ch.QueueBind(
						q.Name,                   // queue name
						bindingConfig.RoutingKey, // routing key
						bindingConfig.Exchange,   // exchange
						false,
						nil)

					if err != nil {
						return err
					}
				}
			}
		}
	}

	return
}

// Close Closes the connection to the rabbitmq server
func (c *MessageConsumer) Close() error {

	err := c.Connection.Close()

	return err
}

//ConnectionString Format an AMQP Connection String
func (config BusConfig) ConnectionString() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s", config.User, config.Password, config.Host, config.Port, config.Vhost)
}
