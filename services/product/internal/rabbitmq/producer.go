package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Producer interface {
	PublishProductCreated(event interface{}) error
	Close()
}

type producer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewProducer(rabbitmqURL string) (Producer, error) {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	err = ch.ExchangeDeclare(
		"product_events", // Name
		"fanout",         // Type
		true,             // Durable
		false,            // Auto-deleted
		false,            // Internal
		false,            // No-wait
		nil,              // Arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &producer{conn: conn, channel: ch}, nil
}

func (p *producer) PublishProductCreated(event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		ctx,
		"product_events", // Exchange
		"",               // Routing key (fanout ignores routing key)
		false,            // Mandatory
		false,            // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return err
	}

	log.Printf("[RabbitMQ] Event Published to product_events: %s", string(body))
	return nil
}

func (p *producer) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
