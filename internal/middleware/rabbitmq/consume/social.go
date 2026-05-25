package consume

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wtitdn/renew_video/internal/entity"
	"github.com/wtitdn/renew_video/internal/middleware/rabbitmq/event"
	"github.com/wtitdn/renew_video/internal/repo"
	"github.com/wtitdn/renew_video/pkg/rabbitmq"
)

type SocialWorker struct {
	ch    *amqp.Channel
	repo  *repo.SocialRepository
	queue string
}

func NewSocialWorker(ch *amqp.Channel, repo *repo.SocialRepository, queue string) *SocialWorker {
	return &SocialWorker{ch: ch, repo: repo, queue: queue}
}

func (w *SocialWorker) Run(ctx context.Context) error {
	if w == nil || w.ch == nil || w.repo == nil {
		return errors.New("social worker is not initialized")
	}
	if w.queue == "" {
		return errors.New("queue is required")
	}
	return runConsumer(ctx, w.ch, w.queue, w.handleDelivery)
}

func (w *SocialWorker) handleDelivery(ctx context.Context, d amqp.Delivery) {
	if err := w.process(ctx, d.Body); err != nil {
		retryCount := rabbitmq.GetRetryCount(d)
		if retryCount >= rabbitmq.MaxRetryCount {
			log.Printf("social worker: max retries exceeded (%d), moving to DLX: %v", retryCount, err)
			_ = d.Ack(false)
			return
		}
		log.Printf("social worker: failed (retry %d/%d): %v", retryCount+1, rabbitmq.MaxRetryCount, err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

func (w *SocialWorker) process(ctx context.Context, body []byte) error {
	var evt event.SocialEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		// 解析事件失败，直接丢弃
		return nil
	}
	if evt.FollowerID == 0 || evt.VloggerID == 0 {
		return nil
	}

	switch evt.Action {
	case "follow":
		err := w.repo.Follow(ctx, &entity.Social{
			FollowerID: evt.FollowerID,
			VloggerID:  evt.VloggerID,
		})
		if err == nil {
			return nil
		}
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil
		}
		return err
	case "unfollow":
		return w.repo.Unfollow(ctx, &entity.Social{
			FollowerID: evt.FollowerID,
			VloggerID:  evt.VloggerID,
		})
	default:
		return nil
	}
}
