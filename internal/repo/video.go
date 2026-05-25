package repo

import (
	"context"
	"errors"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type VideoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}
func (vr *VideoRepository) CreateMsg(ctx context.Context, Msg *entity.OutboxMsg) error {
	if err := vr.db.WithContext(ctx).Create(Msg).Error; err != nil {
		return err
	}
	return nil
}
func (vr *VideoRepository) UpdateLikesCount(ctx context.Context, id uint, likesCount int64) error {
	if err := vr.db.WithContext(ctx).Model(&entity.Video{}).
		Where("id = ?", id).
		Update("likes_count", likesCount).Error; err != nil {
		return err
	}
	return nil
}
func (vr *VideoRepository) IsExist(ctx context.Context, id uint) (bool, error) {
	var video entity.Video
	if err := vr.db.WithContext(ctx).First(&video, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
func (vr *VideoRepository) UpdatePopularity(ctx context.Context, id uint, change int64) error {
	if err := vr.db.WithContext(ctx).Model(&entity.Video{}).
		Where("id = ?", id).
		Update("popularity", gorm.Expr("popularity + ?", change)).Error; err != nil {
		return err
	}
	return nil
}
func (vr *VideoRepository) ChangeLikesCount(ctx context.Context, id uint, change int64) error {
	if err := vr.db.WithContext(ctx).Model(&entity.Video{}).
		Where("id = ?", id).
		UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count + ?, 0)", change)).Error; err != nil {
		return err
	}
	return nil
}
func (vr *VideoRepository) ChangePopularity(ctx context.Context, id uint, change int64) error {
	if err := vr.db.WithContext(ctx).Model(&entity.Video{}).
		Where("id = ?", id).
		UpdateColumn("popularity", gorm.Expr("GREATEST(popularity + ?, 0)", change)).Error; err != nil {
		return err
	}
	return nil

}
func (vr *VideoRepository) CountByAuthor(ctx context.Context, authorID uint) (int64, error) {
	var count int64
	if err := vr.db.WithContext(ctx).Model(&entity.Video{}).Where("author_id = ?", authorID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil

}
func (vr *VideoRepository) TotalLikesByAuthor(ctx context.Context, authorID uint) (int64, error) {
	var total int64
	if err := vr.db.WithContext(ctx).Model(&entity.Video{}).Where("author_id = ?", authorID).Select("COALESCE(SUM(likes_count), 0)").Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (vr *VideoRepository) ListByAuthorID(ctx context.Context, authorID int64) ([]entity.Video, error) {
	var videos []entity.Video
	if err := vr.db.WithContext(ctx).
		Where("author_id = ?", authorID).
		Order("create_time desc").
		Limit(200).
		Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}
func (vr *VideoRepository) GetByID(ctx context.Context, id uint) (*entity.Video, error) {
	var video entity.Video
	if err := vr.db.WithContext(ctx).
		Where("id = ?", id).
		Find(&video).Error; err != nil {
		return nil, err
	}
	return &video, nil
}
func (r *VideoRepository) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// 通过minio重构部分
func (vr *VideoRepository) CreateVideo(ctx context.Context, video *entity.Video) error {
	return vr.db.WithContext(ctx).Create(video).Error
}

func (vr *VideoRepository) DeleteVideo(ctx context.Context, id uint) error {
	return vr.db.WithContext(ctx).Delete(&entity.Video{}, id).Error
}
