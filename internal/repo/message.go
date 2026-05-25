package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type MesRepository struct{ db *gorm.DB }

func NewMesRepository(db *gorm.DB) *MesRepository { return &MesRepository{db: db} }

func (r *MesRepository) AutoMigrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&entity.Message{})
}

func (r *MesRepository) Send(ctx context.Context, m *entity.Message) error {
	m.Content = strings.TrimSpace(m.Content)
	if m.Content == "" {
		return errors.New("content is required")
	}
	m.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *MesRepository) List(ctx context.Context, userID, peerID uint, limit int) ([]entity.Message, error) {
	var msgs []entity.Message
	err := r.db.WithContext(ctx).
		Where("(from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?)", userID, peerID, peerID, userID).
		Order("created_at desc").
		Limit(limit).
		Find(&msgs).Error
	return msgs, err
}
