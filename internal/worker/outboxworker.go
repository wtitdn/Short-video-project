package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	oredis "github.com/redis/go-redis/v9"
	"github.com/wtitdn/renew_video/internal/entity"
	"github.com/wtitdn/renew_video/internal/middleware/rabbitmq/event"
	"github.com/wtitdn/renew_video/internal/middleware/rabbitmq/producer"
	"github.com/wtitdn/renew_video/pkg/redis"
	"gorm.io/gorm"
)

// 从数据库中不断取状态为提交中的数据，放进队列中
func StartOutboxPoller(db *gorm.DB, tmq *producer.TimelineMQ) {
	if db == nil || tmq == nil || tmq.RabbitMQ == nil || tmq.Ch == nil {
		log.Printf("Outbox poller disabled: timeline mq is not initialized")
		return
	}

	go func() {
		for {
			var messages []entity.OutboxMsg

			err := db.Where("status = ?", "pending").Order("create_time ASC").Limit(100).Find(&messages).Error

			if err != nil || len(messages) == 0 {
				time.Sleep(1 * time.Second)
				continue
			}

			for _, msg := range messages {
				err := tmq.PublishVideo(context.Background(), msg.VideoID, msg.CreateTime)

				if err == nil {
					//拿出来了，从数据库中删除
					db.Delete(&msg)
				} else {
					log.Printf("投递MQ失败: VideoID: %d, err: %v", msg.VideoID, err)
				}
			}
		}
	}()
}

func StartConsumer(tmq *producer.TimelineMQ, queueName string, redisClient *redis.Client) {
	if tmq == nil || tmq.RabbitMQ == nil || tmq.Ch == nil {
		log.Printf("Timeline consumer disabled: timeline mq is not initialized")
		return
	}
	if redisClient == nil {
		log.Printf("Timeline consumer disabled: redis is not initialized")
		return
	}

	msgs, err := tmq.Ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		log.Printf("注册消费失败")
		return
	}

	go func() {
		for msg := range msgs {
			var evt event.TimelineEvent
			err := json.Unmarshal(msg.Body, &evt)

			if err != nil {
				log.Printf("反序列化失败")
				msg.Ack(false)
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			timelineKey := redisClient.Key("feed:global_timeline")
			//写入redis
			err = redisClient.ZAdd(ctx, timelineKey, oredis.Z{
				Score:  float64(evt.CreateTime),
				Member: fmt.Sprintf("%d", evt.VideoID),
			})

			if err != nil {
				log.Printf("写入Zset失败")
				//写入失败，重新放回队列
				msg.Nack(false, true)
				cancel()
				continue
			}

			err = redisClient.ZRemRangeByRank(ctx, timelineKey, 0, -1001)

			if err != nil {
				log.Printf("ZRem失败")
			}
			//这条消息没有处理失败，踢出队列
			msg.Ack(false)
			cancel()
		}
	}()
}
