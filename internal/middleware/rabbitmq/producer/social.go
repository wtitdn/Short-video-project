package producer

import (
	"context"
	"errors"
	"time"

	"github.com/wtitdn/renew_video/internal/middleware/rabbitmq/event"
	mqrabbit "github.com/wtitdn/renew_video/pkg/rabbitmq"
)

type SocialMQ struct {
	*mqrabbit.RabbitMQ
}

const (
	socialExchange   = "social.events"
	socialQueue      = "social.events"
	socialBindingKey = "social.*"

	socialFollowRK   = "social.follow"
	socialUnfollowRK = "social.unfollow"
)

func NewSocialMQ(base *mqrabbit.RabbitMQ) (*SocialMQ, error) {
	if base == nil {
		return nil, errors.New("invalid base social middleware")
	}
	if err := base.DeclareTopic(socialExchange, socialQueue, socialBindingKey); err != nil {
		return nil, err
	}
	return &SocialMQ{base}, nil
}

func (s *SocialMQ) publish(ctx context.Context, action, routingKey string, followerID, vloggerID uint) error {
	if s == nil || s.RabbitMQ == nil {
		return errors.New("social mq is not initialized")
	}
	if followerID == 0 || vloggerID == 0 {
		return errors.New("followerID and vloggerID are required")
	}
	id, err := newEventID(16)
	if err != nil {
		return err
	}
	evt := event.SocialEvent{
		EventID:    id,
		Action:     action,
		FollowerID: followerID,
		VloggerID:  vloggerID,
		OccurredAt: time.Now().UTC(),
	}
	return s.PublishJSON(ctx, socialExchange, routingKey, evt)
}
func (s *SocialMQ) Follow(ctx context.Context, followerID, vloggerID uint) error {
	return s.publish(ctx, "follow", socialFollowRK, followerID, vloggerID)
}

func (s *SocialMQ) UnFollow(ctx context.Context, followerID, vloggerID uint) error {
	return s.publish(ctx, "unfollow", socialUnfollowRK, followerID, vloggerID)
}
