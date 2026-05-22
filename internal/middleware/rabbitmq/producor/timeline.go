package producor

import (
	"context"
	"errors"
	"time"

	"github.com/wtitdn/renew_video/internal/middleware/rabbitmq/rbentity"
	mqrabbit "github.com/wtitdn/renew_video/pkg/rabbitmq"
)

type TimelineMQ struct {
	*mqrabbit.RabbitMQ
}

const (
	timelineExchange   = "video.timeline.events"
	timelineQueue      = "video.timeline.update.queue"
	timelineBindingKey = "video.timeline.*"
	timelinePublishRK  = "video.timeline.publish"
)

func NewTimelineMQ(base *mqrabbit.RabbitMQ) (*TimelineMQ, error) {
	if base == nil {
		return nil, errors.New("nil TimelineMQ")
	}
	if err := base.DeclareTopic(timelineExchange, timelineQueue, timelineBindingKey); err != nil {
		return nil, err
	}
	return &TimelineMQ{base}, nil
}

func (t *TimelineMQ) PublishVideo(ctx context.Context, videoID uint, createTime time.Time) error {
	if t == nil || t.RabbitMQ == nil {
		return errors.New("timeline mq is not initialized")
	}
	if videoID == 0 {
		return errors.New("videoID are required")
	}
	id, err := newEventID(16)
	if err != nil {
		return err
	}
	timeline := rbentity.TimelineEvent{
		EventID:    id,
		VideoID:    videoID,
		CreateTime: createTime.UnixMilli(),
		OccurredAt: time.Now(),
	}
	return t.PublishJSON(ctx, timelineExchange, timelinePublishRK, timeline)
}
