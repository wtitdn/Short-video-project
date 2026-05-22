package producer

import (
	"context"
	"errors"
	"time"

	"github.com/wtitdn/renew_video/internal/middleware/rabbitmq/event"
	mqrabbit "github.com/wtitdn/renew_video/pkg/rabbitmq"
)

type PopularityMQ struct {
	*mqrabbit.RabbitMQ
}

const (
	popularityExchange   = "video.popularity.events"
	popularityQueue      = "video.popularity.events"
	popularityBindingKey = "video.popularity.*"

	popularityUpdateRK = "video.popularity.update"
)

func Newpopularity(base *mqrabbit.RabbitMQ) (*PopularityMQ, error) {
	if base == nil {
		return nil, errors.New("mq is nil")
	}
	if err := base.DeclareTopic(popularityExchange, popularityQueue, popularityBindingKey); err != nil {
		return nil, err
	}
	return &PopularityMQ{base}, nil
}

// 不需要publish 因为这个只是实时返回热度榜单
func (p *PopularityMQ) Update(ctx context.Context, videoID uint, change int64) error {
	if p == nil || p.RabbitMQ == nil {
		return errors.New("popularity mq is not initialized")
	}
	if videoID == 0 || change == 0 {
		return errors.New("videoID and change are required")
	}
	id, err := newEventID(16)
	if err != nil {
		return err
	}
	evt := event.PopularityEvent{
		EventID:    id,
		VideoID:    videoID,
		Change:     change,
		OccurredAt: time.Now().UTC(),
	}
	return p.PublishJSON(ctx, popularityExchange, popularityUpdateRK, evt)
}
