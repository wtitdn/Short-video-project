package producor

import (
	"context"
	"errors"
	"time"

	mqentity "github.com/wtitdn/renew_video/internal/middleware/rabbitmq/rbentity"
	mqrabbit "github.com/wtitdn/renew_video/pkg/rabbitmq"
)

type LikeMQ struct {
	*mqrabbit.RabbitMQ
}

const (
	likeExchange   = "like.events"
	likeQueue      = "like.events"
	likeBindingKey = "like.*"

	likeLikeRK   = "like.like"
	likeUnlikeRK = "like.unlike"
)

// 建立like的生产者模型和队列
func NewLikeMQ(base *mqrabbit.RabbitMQ) (*LikeMQ, error) {
	if base == nil {
		return nil, errors.New("mqrabbit base is nil")
	}
	if err := base.DeclareTopic(likeExchange, likeQueue, likeBindingKey); err != nil {
		return nil, err
	}
	return &LikeMQ{base}, nil
}
func (l *LikeMQ) pulish(ctx context.Context, action, routingKey string, userID, videoID uint) error {
	if l == nil || l.RabbitMQ == nil {
		return errors.New("like mq isn't initialized")
	}
	if userID == 0 || videoID == 0 {
		return errors.New("userID and videoID are required")
	}
	id, err := newEventID(16)
	if err != nil {
		return err
	}
	event := mqentity.LikeEvent{
		EventID:    id,
		Action:     action,
		UserID:     userID,
		VideoID:    videoID,
		OccurredAt: time.Now(),
	}
	return l.PublishJSON(ctx, likeExchange, routingKey, event)
}
func (l *LikeMQ) Like(ctx context.Context, userid, videoId uint) error {
	return l.pulish(ctx, "like", likeLikeRK, userid, videoId)
}
func (l *LikeMQ) Unlike(ctx context.Context, userid, videoId uint) error {
	return l.pulish(ctx, "unlike", likeUnlikeRK, userid, videoId)
}
