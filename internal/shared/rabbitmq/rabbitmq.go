package rabbitmq

import (
	"github.com/rabbitmq/amqp091-go"
	"log"
	"os"
)

type RabbitMQ struct {
	Conn    *amqp091.Connection
	Channel *amqp091.Channel
}

func New() *RabbitMQ {
	conn, err := amqp091.Dial(os.Getenv("RABBITMQ_HOST"))
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
