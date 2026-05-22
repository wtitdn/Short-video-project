package producor

import (
	"context"
	"errors"
	"time"

	mqentity "github.com/wtitdn/renew_video/internal/middleware/rabbitmq/rbentity"
	mqrabbit "github.com/wtitdn/renew_video/pkg/rabbitmq"
)

type CommentMQ struct {
	*mqrabbit.RabbitMQ
}

const (
	commentExchange   = "comment.events"
	commentQueue      = "comment.events"
	commentBindingKey = "comment.*"

	commentPublishRK = "comment.publish"
	commentDeleteRK  = "comment.delete"
)

func NewCommentMQ(base *mqrabbit.RabbitMQ) (*CommentMQ, error) {
	if base == nil {
		return nil, errors.New("rabbitmq base is nil")
	}
	//创建生产者和交换机一集队列
	if err := base.DeclareTopic(commentExchange, commentQueue, commentBindingKey); err != nil {
		return nil, err
	}
	return &CommentMQ{RabbitMQ: base}, nil
}

// 发布视频消息
func (c *CommentMQ) Publish(ctx context.Context, username string, videoID, authorID uint, content string) error {
	return c.publish(ctx, "publish", commentPublishRK, mqentity.CommentEvent{
		Username: username,
		VideoID:  videoID,
		AuthorID: authorID,
		Content:  content,
	})
}

// 删除视频消息
func (c *CommentMQ) Delete(ctx context.Context, commentID uint) error {
	return c.publish(ctx, "delete", commentDeleteRK, mqentity.CommentEvent{
		CommentID: commentID,
	})
}

// 统一发布消息的组件
func (c *CommentMQ) publish(ctx context.Context, action, routingKey string, evt mqentity.CommentEvent) error {
	if c == nil || c.RabbitMQ == nil {
		return errors.New("comment mq is not initialized")
	}
	id, err := newEventID(16)
	if err != nil {
		return err
	}
	evt.EventID = id
	evt.Action = action
	evt.OccurredAt = time.Now().UTC()
	return c.PublishJSON(ctx, commentExchange, routingKey, evt)
}
