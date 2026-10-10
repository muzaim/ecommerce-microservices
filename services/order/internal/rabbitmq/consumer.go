package rabbitmq

import (
	"encoding/json"
	"log"
	"time"

	"ecommerce/services/order/internal/repository"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SagaStockResponseEvent struct {
	Event   string `json:"event"`
	OrderID uint   `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

func StartConsumer(rabbitmqURL string, orderRepo repository.OrderRepository) {
	go func() {
		for {
			err := listen(rabbitmqURL, orderRepo)
			if err != nil {
				log.Printf("[Saga Consumer Error] %v. Retrying in 5 seconds...", err)
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

func listen(rabbitmqURL string, orderRepo repository.OrderRepository) error {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"product_events",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	q, err := ch.QueueDeclare(
		"order_service_saga_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = ch.QueueBind(
		q.Name,
		"",
		"product_events",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Println("[Saga Consumer] Connected & Listening for Stock Saga responses...")

	for d := range msgs {
		var event SagaStockResponseEvent
		if err := json.Unmarshal(d.Body, &event); err != nil {
			continue
		}

		switch event.Event {
		case "StockReserved":
			log.Printf("[Saga Success] Stock Reserved for Order ID %d. Updating status to CONFIRMED", event.OrderID)
			if err := orderRepo.UpdateOrderStatus(event.OrderID, "CONFIRMED"); err != nil {
				log.Printf("[Saga Error] Failed to update order %d to CONFIRMED: %v", event.OrderID, err)
			}

		case "StockFailed":
			log.Printf("[Saga Rollback] Stock Failed for Order ID %d (%s). Updating status to CANCELLED", event.OrderID, event.Reason)
			if err := orderRepo.UpdateOrderStatus(event.OrderID, "CANCELLED"); err != nil {
				log.Printf("[Saga Error] Failed to update order %d to CANCELLED: %v", event.OrderID, err)
			}
		}
	}

	return nil
}
