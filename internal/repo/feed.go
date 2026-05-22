package repo

import (
	"context"
	"time"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type FeedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository { return &FeedRepository{db: db} }

func (repo *FeedRepository) ListLatest(ctx context.Context, limit int, latestBefore time.Time) ([]*entity.Video, error) {
	var videos []*entity.Video
	query := repo.db.WithContext(ctx).Model(&entity.Video{}).
		Order("create_time DESC")
	if !latestBefore.IsZero() {
		query = query.Where("create_time < ?", latestBefore)
	}
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}
func (repo *FeedRepository) ListLikesCountWithCursor(ctx context.Context, limit int, cursor *entity.LikesCountCursor) ([]*entity.Video, error) {
	videos := []*entity.Video{}
	query := repo.db.WithContext(ctx).Model(&entity.Video{}).
		Order("likes_count DESC, id DESC")
	if cursor != nil {
		query = query.Where(
			"(likes_count < ?) OR (likes_count = ? AND id < ?)",
			cursor.LikesCount,
			cursor.LikesCount, cursor.ID,
		)
	}
	//结果赋值 在这里对videos进行赋值的
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil

}
func (repo *FeedRepository) ListByFollowing(ctx context.Context, limit int, viewerAccountID uint, latestBefore time.Time) ([]*entity.Video, error) {
	videos := []*entity.Video{}
	query := repo.db.WithContext(ctx).Model(&entity.Video{}).
		Order("create_time DESC")
	if viewerAccountID > 0 {
		followingSubQuery := repo.db.WithContext(ctx).
			Model(&entity.Social{}).
			Select("vlogger_id").
			Where("follower_id = ?", viewerAccountID)
		query = query.Where("author_id IN (?)", followingSubQuery)
	}
	if !latestBefore.IsZero() {
		query = query.Where("create_time < ?", latestBefore)
	}
	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}
func (repo *FeedRepository) ListByPopularity(ctx context.Context, limit int, popularityBefore int64, timeBefore time.Time, idBefore uint) ([]*entity.Video, error) {
	videos := []*entity.Video{}
	query := repo.db.WithContext(ctx).Model(&entity.Video{}).
		Order("popularity DESC, create_time DESC, id DESC")
	// 只有当游标完整提供时才加过滤（popularity 允许为 0）
	if !timeBefore.IsZero() && idBefore > 0 {
		query = query.Where(
			"(popularity < ?) OR (popularity = ? AND create_time < ?) OR (popularity = ? AND create_time = ? AND id < ?)",
			popularityBefore,
			popularityBefore, timeBefore,
			popularityBefore, timeBefore, idBefore,
		)
	}

	if err := query.Limit(limit).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

// 根据id查视频
func (repo *FeedRepository) GetByIDs(ctx context.Context, ids []uint) ([]*entity.Video, error) {
	videos := []*entity.Video{}
	if len(ids) == 0 {
		return videos, nil
	}
	if err := repo.db.WithContext(ctx).Model(&entity.Video{}).
		Where("id IN ?", ids).Find(&videos).Error; err != nil {

		return nil, err
	}
	return videos, nil
}
func (repo *FeedRepository) ListByTag(ctx context.Context, tagName string, limit int) ([]*entity.Video, error) {
	var videos []*entity.Video
	//join语法 join a on b = 条件，将a和b两张表根据条件关联在一起
	// order 根据value值进行条件排序

	err := repo.db.WithContext(ctx).Model(&entity.Video{}).Table("videos").
		Joins("JOIN video_tags ON video_tags.video_id = videos.id").
		Joins("JOIN tags ON tags.id = video_tags.tag_id").
		Where("tags.name = ?", tagName).
		Order("videos.create_time desc").
		Limit(limit).
		Find(&videos).Error
	return videos, err
}
