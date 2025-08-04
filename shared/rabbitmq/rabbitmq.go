package rabbitmq

import (
	"github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

type RabbitMQ struct {
	Conn      *amqp091.Connection
	Channel   *amqp091.Channel
	mu        *sync.Mutex
	connected atomic.Bool
}

// TODO store messages to disk and replay if connection is not working
// TODO connection and error handling, retries
// TODO circuit breaker, only expose some methods and not the channel/conn

func New() *RabbitMQ {
	conn, err := amqp091.DialConfig(os.Getenv("RABBITMQ_HOST"), amqp091.Config{
		Heartbeat: 10,
		Locale:    "en_US",
	})
	if err != nil {
		panic(err)
	}
	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	log.Default().Println("Connected to RabbitMQ")
	return &RabbitMQ{
		Conn:    conn,
		Channel: ch,
	}
}
func (r *RabbitMQ) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Conn.Close()
}
