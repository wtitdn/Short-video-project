package entity

import (
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wtitdn/renew_video/internal/repo"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
	"gorm.io/gorm"
)

type CommentWorker struct {
	ch       *amqp.Channel
	comments *repo.CommentRepository
	videos   *repo.VideoRepository
	queue    string
}

type CommentEvent struct {
	EventID    string    `json:"event_id"`
	Action     string    `json:"action"`
	CommentID  uint      `json:"comment_id,omitempty"`
	Username   string    `json:"username,omitempty"`
	VideoID    uint      `json:"video_id,omitempty"`
	AuthorID   uint      `json:"author_id,omitempty"`
	Content    string    `json:"content,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}
type LikeWorker struct {
	ch     *amqp.Channel
	likes  *repo.LikeRepository
	videos *repo.VideoRepository
	queue  string
}

type WorkerNotification struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	RecipientID uint      `gorm:"index;not null" json:"recipient_id"`
	SenderID    uint      `gorm:"not null" json:"sender_id"`
	Type        string    `gorm:"type:varchar(50);not null" json:"type"`
	TargetID    uint      `json:"target_id"`
	Content     string    `gorm:"type:varchar(255)" json:"content"`
	IsRead      bool      `gorm:"default:false" json:"is_read"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type PopularityWorker struct {
	ch    *amqp.Channel
	cache *rediscache.Client
	queue string
}

type SocialWorker struct {
	ch    *amqp.Channel
	repo  *repo.SocialRepository
	queue string
}
type SSEHub struct {
	mu      sync.RWMutex
	clients map[uint][]chan *WorkerNotification
	db      *gorm.DB
}
