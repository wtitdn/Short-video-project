package consume

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wtitdn/renew_video/internal/entity"
	"github.com/wtitdn/renew_video/internal/middleware/rabbitmq/event"
	"github.com/wtitdn/renew_video/internal/repo"
	"github.com/wtitdn/renew_video/pkg/rabbitmq"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
)

type LikeWorker struct {
	ch     *amqp.Channel
	likes  *repo.LikeRepository
	videos *repo.VideoRepository
	cache  *rediscache.Client
	queue  string
}

func NewLikeWorker(ch *amqp.Channel, likes *repo.LikeRepository, videos *repo.VideoRepository, cache *rediscache.Client, queue string) *LikeWorker {
	return &LikeWorker{ch: ch, likes: likes, videos: videos, cache: cache, queue: queue}
}

func (w *LikeWorker) invalidateVideoDetailCache(videoID uint) {
	if w.cache == nil || videoID == 0 {
		return
	}
	_ = w.cache.Del(context.Background(), w.cache.Key("video:detail:id=%d", videoID))
}

// 接受到中继器的信息
func (w *LikeWorker) handleDelivery(ctx context.Context, d amqp.Delivery) {
	if err := w.process(ctx, d.Body); err != nil {
		retryCount := rabbitmq.GetRetryCount(d)
		if retryCount >= rabbitmq.MaxRetryCount {
			log.Printf("like worker: max retries exceeded (%d), moving to DLX: %v", retryCount, err)
			_ = d.Ack(false)
			return
		}
		log.Printf("like worker: failed (retry %d/%d): %v", retryCount+1, rabbitmq.MaxRetryCount, err)
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

//处理函数

func (w *LikeWorker) process(ctx context.Context, body []byte) error {
	evt := event.LikeEvent{}
	if err := json.Unmarshal(body, &evt); err != nil {
	}
	if evt.UserID == 0 || evt.VideoID == 0 {
		return nil
	}
	//决定是什么路径执行
	switch evt.Action {
	case "like":
		return w.applyLike(ctx, evt.UserID, evt.VideoID)
	case "unlike":
		return w.applyUnlike(ctx, evt.UserID, evt.VideoID)
	default:
		return nil
	}
}

// like
func (w *LikeWorker) applyLike(ctx context.Context, userID, videoID uint) error {
	ok, err := w.videos.IsExist(ctx, videoID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	created, err := w.likes.LikeIgnoreDuplicate(ctx, &entity.Like{
		VideoID:   videoID,
		AccountID: userID,
		CreatedAt: time.Now(),
	})
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	if err := w.videos.ChangeLikesCount(ctx, videoID, 1); err != nil {
		return err
	}
	if err := w.videos.ChangePopularity(ctx, videoID, 1); err != nil {
		return err
	}
	w.invalidateVideoDetailCache(videoID)
	return nil
}
func (w *LikeWorker) applyUnlike(ctx context.Context, userID, videoID uint) error {
	ok, err := w.videos.IsExist(ctx, videoID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	deleted, err := w.likes.DeleteByVideoAndAccount(ctx, videoID, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return nil
	}

	if err := w.videos.ChangeLikesCount(ctx, videoID, -1); err != nil {
		return err
	}
	if err := w.videos.ChangePopularity(ctx, videoID, -1); err != nil {
		return err
	}
	w.invalidateVideoDetailCache(videoID)
	return nil
}
func (w *LikeWorker) Run(ctx context.Context) error {
	if w == nil || w.ch == nil || w.likes == nil || w.videos == nil {
		return errors.New("comment worker is not initialized")
	}
	if w.queue == "" {
		return errors.New("queue is required")
	}

	return runConsumer(ctx, w.ch, w.queue, w.handleDelivery)
}
