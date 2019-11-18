package directdebit

import (
	"fmt"

	"github.com/streadway/amqp"
)

//BusConfig RabbitMQ configuration definition
type BusConfig struct {
	User string `json:"user"`
	// Password string `json:"password"`
	// Host     string `json:"host"`
	// Port     string `json:"port"`
	// Vhost    string `json:"vhost"`
	// TLS      bool   `json:"tls"`
	//Exchanges []ExchangeConfig `json:"exchanges"`
	//Queues    []QueueConfig    `json:"queues"`
}

//ExchangeConfig RabbbitMQ Exchange configuration
type ExchangeConfig struct {
	Name         string `json:"name"`
	ExchangeType string `json:"type"`
	Durable      bool   `json:"durable"`
}

//QueueConfig RabbitMQ Queue definition
type QueueConfig struct {
	Name           string          `json:"name"`
	Durable        bool            `json:"durable"`
	DeleteOnUnused bool            `json:"deleteOnUnused"`
	Exclusive      bool            `json:"exclusive"`
	NoWait         bool            `json:"noWait"`
	Args           string          `json:"args"`
	Bindings       []BindingConfig `json:"bindings"`
}

//BindingConfig Queue/Exchange Bindings
type BindingConfig struct {
	RoutingKey string `json:"routingKey"`
	Exchange   string `json:"exchange"`
}

//ConnectionString Format an AMQP Connection String
func (config BusConfig) ConnectionString() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/%s", config.User, config.Password, config.Host, config.Port, config.Vhost)
}

func configureMessageBus(conn *amqp.Connection, config *BusConfig) (*amqp.Channel, error) {

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	/*

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
					return nil, err
				}
			}
		}

		if len(config.Queues) > 0 {
			for _, qConfig := range config.Queues {
				q, err := ch.QueueDeclare(
					qConfig.Name,           // name
					qConfig.Durable,        // durable
					qConfig.DeleteOnUnused, // delete when unused
					true,                   // exclusive
					false,                  // no-wait
					nil,                    // arguments
				)
				if err != nil {
					return nil, err
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
							return nil, err
						}
					}
				}
			}
		}

	*/
	return ch, err
}
