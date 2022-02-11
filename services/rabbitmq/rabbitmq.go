package rabbitmq

import (
	"dishrank-go-worker/utils"
	"errors"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

//MessageBody is the struct for the body passed in the AMQP message. The type will be set on the Request header
type MessageBody struct {
	Data []byte
	Type string
}

//Message is the amqp request to publish
type Message struct {
	Queue         string
	ReplyTo       string
	ContentType   string
	CorrelationID string
	Priority      uint8
	Body          MessageBody
}

//Connection is the connection created
type Connection struct {
	ShutdownErr *chan error
	name        string
	Conn        *amqp.Connection
	Channel     *amqp.Channel
	Queues      []string
	Err         chan error
	ApiErr      chan error
}

var (
	connectionPool = make(map[string]*Connection)
)

//NewConnection returns the new connection object
func NewConnection(name string, queues []string) *Connection {
	if c, ok := connectionPool[name]; ok {
		return c
	}
	c := &Connection{
		Queues: queues,
		Err:    make(chan error),
		ApiErr: make(chan error),
	}
	connectionPool[name] = c
	return c
}

//GetConnection returns the connection which was instantiated
func GetConnection(name string) *Connection {
	return connectionPool[name]
}

func (c *Connection) Connect() error {
	var err error
	c.Conn, err = amqp.Dial(utils.EnvConfig.RabbitMQ.Domain)
	if err != nil {
		return fmt.Errorf("Error in creating rabbitmq connection with %s : %s", utils.EnvConfig.RabbitMQ.Domain, err.Error())
	}
	go func() {
		<-c.Conn.NotifyClose(make(chan *amqp.Error)) //Listen to NotifyClose
		c.Err <- errors.New("Connection Closed")
		c.ApiErr <- errors.New("Api detect Connection Closed")
	}()
	c.Channel, err = c.Conn.Channel()
	if err != nil {
		return fmt.Errorf("Channel: %s", err)
	}
	return nil
}

func (c *Connection) BindQueue() error {
	for _, q := range c.Queues {
		if _, err := c.Channel.QueueDeclare(q, false, false, false, false, nil); err != nil {
			return fmt.Errorf("error in declaring the queue %s", err)
		}
	}
	return nil
}

//Reconnect reconnects the connection
func (c *Connection) Reconnect() error {
	if err := c.Connect(); err != nil {
		return err
	}
	if err := c.BindQueue(); err != nil {
		return err
	}
	return nil
}

func (c *Connection) Consume() (map[string]<-chan amqp.Delivery, error) {
	m := make(map[string]<-chan amqp.Delivery)
	for _, q := range c.Queues {
		deliveries, err := c.Channel.Consume(q, "", true, false, false, false, nil)
		if err != nil {
			return nil, err
		}
		m[q] = deliveries
	}
	return m, nil
}

func (c *Connection) HandleConsumedDeliveries(q string, delivery <-chan amqp.Delivery, fn func(Connection, string, <-chan amqp.Delivery)) {
	fmt.Println("[HandleConsumedDeliveries]Delivery received")
	for {
		go fn(*c, q, delivery)
		if err := <-c.Err; err != nil {
			for {
				c.Reconnect()

				deliveries, err := c.Consume()
				if err != nil {
					// panic(err)
					time.Sleep(60 * time.Second)
					fmt.Println("try again")
				} else {
					fmt.Println("try ok")
					delivery = deliveries[q]
					break
				}

			}
		}
	}
}
