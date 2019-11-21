package directdebit

import (
	"errors"
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
	Exchanges []ExchangeConfig
	Queues    []QueueConfig
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
	Durable        bool
	DeleteOnUnused bool
	Exclusive      bool
	NoWait         bool
	Args           string
	Bindings       []BindingConfig `json:"bindings"`
}

//BindingConfig Queue/Exchange Bindings
type BindingConfig struct {
	RoutingKey string
	Exchange   string
}

//MessageConsumer Consumes a message for the pipeline
type MessageConsumer interface {
	Configure() error
	Consume() (<-chan amqp.Delivery, error)
	Close() error
}

type messageConsumer struct {
	config     *BusConfig
	Connection *amqp.Connection
	Channel    *amqp.Channel
	log        *log.Entry
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
func NewConsumer(config *BusConfig, log *log.Entry) (MessageConsumer, error) {

	consumer := &messageConsumer{
		log:    log.WithField("Component", "Consumer"),
		config: config,
	}

	connString := config.ConnectionString()
	consumer.log.Debug(connString)

	conn, err := amqp.Dial(connString)
	if err != nil {
		consumer.log.Errorf("Failed to connect to RabbitMQ : %s ", err.Error())
		return nil, err
	}
	consumer.Connection = conn

	// Open a Channel
	log.Debug("Opening Channel to RabbitMQ")
	consumer.Channel, err = conn.Channel()

	return consumer, err
}

func (c *messageConsumer) Configure() (err error) {

	if len(c.config.Exchanges) > 0 {
		for _, ex := range c.config.Exchanges {
			err = c.Channel.ExchangeDeclare(
				ex.Name,         // name
				ex.ExchangeType, // type
				true,            // durable
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

	if len(c.config.Queues) > 0 {
		for _, qConfig := range c.config.Queues {
			q, err := c.Channel.QueueDeclare(
				qConfig.Name,           // name
				true,                   // durable
				qConfig.DeleteOnUnused, // delete when unused
				true,                   // exclusive
				false,                  // no-wait
				nil,                    // arguments
			)
			if err != nil {
				return err
			}

			if len(qConfig.Bindings) > 0 {
				for _, bindingConfig := range qConfig.Bindings {
					err := c.Channel.QueueBind(
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

func (c *messageConsumer) Consume() (<-chan amqp.Delivery, error) {
	if c.Channel != nil {
		c.log.Debug("Registering Consumer")

		// @Todo this should return slices
		return c.Channel.Consume(
			c.config.Queues[0].Name, // queue
			"pipefire",              // consumer
			true,                    // auto-ack
			false,                   // exclusive
			false,                   // no-local
			false,                   // no-wait
			nil,                     // args
		)
	}
	return nil, errors.New("can't register consumer, channel is not open")
}

func (c *messageConsumer) Close() error {

	err := c.Connection.Close()

	return err
}

//ConnectionString Format an AMQP Connection String
func (config BusConfig) ConnectionString() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s", config.User, config.Password, config.Host, config.Port, config.Vhost)
}
