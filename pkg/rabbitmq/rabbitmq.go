package rabbitmq

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/wtitdn/renew_video/internal/config"

	"log"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn *amqp.Connection
	Ch   *amqp.Channel
}

// 建立连接
func NewRabbitMQ(cfg *config.RabbitMQConfig) (*RabbitMQ, error) {
	if cfg == nil {
		return nil, errors.New("rabbitmq config is nil")
	}
	url := "amqp://" + cfg.Username + ":" + cfg.Password + "@" + cfg.Host + ":" + strconv.Itoa(cfg.Port) + "/"
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return &RabbitMQ{Conn: conn, Ch: ch}, nil
}

// 关闭连接
func (r *RabbitMQ) Close() error {
	if r == nil || r.Ch == nil || r.Conn == nil {
		return nil
	}
	if err := r.Ch.Close(); err != nil {
		return err
	}
	if err := r.Conn.Close(); err != nil {
		return err
	}
	return nil
}

// 声明需要的内容
func (r *RabbitMQ) DeclareTopic(exchange string, queue string, bindingKey string) error {
	if r == nil || r.Ch == nil {
		return errors.New("rabbitmq is not initialized")
	}
	if exchange == "" || queue == "" || bindingKey == "" {
		return errors.New("exchange/queue/bindingKey is required")
	}
	//定义交换机
	if err := r.Ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}
	//定义队列
	q, err := r.Ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		amqp.Table{"x-dead-letter-exchange": DLXExchange},
	)
	if err != nil {
		return err
	}
	//告诉交换机如何发送消息给我们的队列
	if err := r.Ch.QueueBind(
		q.Name,
		bindingKey,
		exchange,
		false,
		nil,
	); err != nil {
		return err
	}
	if err := DeclareDLX(r.Ch, queue); err != nil {
		log.Printf("DLX declare failed for %s: %v", queue, err)
	}
	return nil
}

// 消息发布
func (r *RabbitMQ) PublishJSON(ctx context.Context, exchange string, routingKey string, payload any) error {
	if r == nil || r.Ch == nil {
		return errors.New("rabbitmq is not initialized")
	}
	if exchange == "" || routingKey == "" {
		return errors.New("exchange and routingKey are required")
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return r.Ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         b,
	})
}

func newEventID(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
