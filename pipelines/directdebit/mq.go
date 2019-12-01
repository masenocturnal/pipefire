package directdebit

import (
	"fmt"

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
	Connect()
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

//BusError indicates there is a connection issue with the bus and an action to take
type BusError struct {
	Msg    string
	Action string
}

//NewConsumer provides an instance of MessageConsumer
func NewConsumer(config *BusConfig, log *log.Entry) *MessageConsumer {

	consumer := &MessageConsumer{
		log:    log.WithField("Component", "Consumer"),
		config: config,
	}
	return consumer
}

//Connect Establishes a connection to rabbitmq
func (c *MessageConsumer) Connect() (*amqp.Connection, error) {

	var err error
	uri := c.config.ConnectionString()
	c.log.Info("Try and connect to RabbitMQ")
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}

	c.log.Info("Connected")
	return conn, err
}

//Configure Creates the Exchange and Queue
func (c *MessageConsumer) Configure(ch *amqp.Channel) (err error) {

	config := c.config

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
