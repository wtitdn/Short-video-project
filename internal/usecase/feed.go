package usecase

import (
	"context"
	"time"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type FeedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{db: db}
}

func (repo *FeedRepository) ListLatest(ctx context.Context, limit int, latestBefore time.Time) ([]*entity.Video, error) {

}
func (repo *FeedRepository) ListLikesCountWithCursor(ctx context.Context, limit int, cursor *entity.LikesCountCursor) ([]*entity.Video, error) {

}
func (repo *FeedRepository) ListByFollowing(ctx context.Context, limit int, viewerAccountID uint, latestBefore time.Time) ([]*entity.Video, error) {

}
func (repo *FeedRepository) ListByPopularity(ctx context.Context, limit int, popularityBefore int64, timeBefore time.Time, idBefore uint) ([]*entity.Video, error) {

}

// 获取某个用户所有的视频
func (repo *FeedRepository) GetByIDs(ctx context.Context, ids []uint) ([]*entity.Video, error) {

}

// 获取某个tag的所有视频
func (repo *FeedRepository) ListByTag(ctx context.Context, tagName string, limit int) ([]*entity.Video, error) {

}
