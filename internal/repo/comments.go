package repo

import (
	"context"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) CreateComment(ctx context.Context, comment *entity.Comment) error {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return err
	}
	return nil
}

func (r *CommentRepository) DeleteComment(ctx context.Context, comment *entity.Comment) error {
	if err := r.db.WithContext(ctx).Delete(comment).Error; err != nil {
		return err
	}
	return nil
}

func (r *CommentRepository) GetAllComments(ctx context.Context, videoID uint) ([]entity.Comment, error) {
	var comments []entity.Comment
	err := r.db.WithContext(ctx).Where("video_id = ?", videoID).Order("created_at DESC").Limit(200).Find(&comments).Error
	return comments, err
}

func (r *CommentRepository) IsExist(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Comment{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *CommentRepository) GetByID(ctx context.Context, id uint) (*entity.Comment, error) {
	var comment entity.Comment
	err := r.db.WithContext(ctx).First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *CommentRepository) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *CommentRepository) CreateNotification(ctx context.Context, notif *entity.Notification) error {
	if notif == nil {
		return nil
	}
	return r.db.WithContext(ctx).Create(notif).Error
}
