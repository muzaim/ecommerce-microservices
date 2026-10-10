package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ecommerce/services/product/internal/repository"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type OrderCreatedItem struct {
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}

type OrderCreatedEvent struct {
	Event   string             `json:"event"`
	OrderID uint               `json:"order_id"`
	UserID  uint               `json:"user_id"`
	Items   []OrderCreatedItem `json:"items"`
}

func StartConsumer(rabbitmqURL string, repo repository.ProductRepository, producer Producer, rdb *redis.Client) {
	go func() {
		for {
			err := listen(rabbitmqURL, repo, producer, rdb)
			if err != nil {
				log.Printf("[Product Saga Consumer Error] %v. Retrying in 5 seconds...", err)
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

func listen(rabbitmqURL string, repo repository.ProductRepository, producer Producer, rdb *redis.Client) error {
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
		"order_events",
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
		"product_service_order_queue",
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
		"order_events",
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

	log.Println("[Product Saga Consumer] Connected & Listening for OrderCreated events...")

	for d := range msgs {
		var event OrderCreatedEvent
		if err := json.Unmarshal(d.Body, &event); err != nil {
			continue
		}

		if event.Event == "OrderCreated" {
			log.Printf("[Product Saga Consumer] Processing OrderCreated for Order ID %d", event.OrderID)

			var stockErr error
			for _, item := range event.Items {
				updatedProd, err := repo.DeductStock(item.ProductID, item.Quantity)
				if err != nil {
					stockErr = err
					break
				}

				// IMMEDIATE CACHE UPDATE IN REDIS (0.001ms Real-time Cache Sync)
				if rdb != nil && updatedProd != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					key := fmt.Sprintf("product:%d", updatedProd.ID)
					productJSON, _ := json.Marshal(updatedProd)
					rdb.Set(ctx, key, string(productJSON), 24*time.Hour)
					cancel()
					log.Printf("[Real-time Cache Sync] Updated Redis product:%d stock to %d after order %d",
						updatedProd.ID, updatedProd.Stock, event.OrderID)
				}
			}

			if stockErr != nil {
				log.Printf("[Saga Failure] Failed to reserve stock for Order ID %d: %v", event.OrderID, stockErr)
				failedEvent := map[string]interface{}{
					"event":    "StockFailed",
					"order_id": event.OrderID,
					"reason":   stockErr.Error(),
				}
				producer.PublishProductCreated(failedEvent)
			} else {
				log.Printf("[Saga Success] Stock Reserved successfully for Order ID %d", event.OrderID)
				successEvent := map[string]interface{}{
					"event":    "StockReserved",
					"order_id": event.OrderID,
				}
				producer.PublishProductCreated(successEvent)
			}
		}
	}

	return nil
}
