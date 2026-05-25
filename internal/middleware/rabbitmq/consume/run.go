package consume

import (
	"context"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

type deliveryHandler func(context.Context, amqp.Delivery)

func runConsumer(ctx context.Context, ch *amqp.Channel, queue string, handle deliveryHandler) error {
	if ch == nil {
		return errors.New("rabbitmq channel is nil")
	}
	if queue == "" {
		return errors.New("queue is required")
	}
	if handle == nil {
		return errors.New("delivery handler is nil")
	}

	deliveries, err := ch.Consume(
		queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("deliveries channel closed")
			}
			handle(ctx, d)
		}
	}
}
