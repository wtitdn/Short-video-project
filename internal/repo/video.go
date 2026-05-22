package repo

import (
	"context"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type VideoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) *VideoRepository {
	return &VideoRepository{db: db}
}
func (vr *VideoRepository) CreateVideo(ctx context.Context, video *entity.Video) error {

}

func (vr *VideoRepository) DeleteVideo(ctx context.Context, id string) error {

}
func (vr *VideoRepository) CreateMsg(ctx context.Context, Msg *entity.OutboxMsg) error {

}
func (vr *VideoRepository) UpdateLikesCount(ctx context.Context, id uint, likesCount int64) error {

}
func (vr *VideoRepository) IsExist(ctx context.Context, id uint) (bool, error) {

}
func (vr *VideoRepository) UpdatePopularity(ctx context.Context, id uint, change int64) error {

}
func (vr *VideoRepository) ChangeLikesCount(ctx context.Context, id uint, change int64) error {

}
func (vr *VideoRepository) ChangePopularity(ctx context.Context, id uint, change int64) error {

}
func (vr *VideoRepository) CountByAuthor(ctx context.Context, authorID uint) (int64, error) {

}
func (vr *VideoRepository) TotalLikesByAuthor(ctx context.Context, authorID uint) (int64, error) {

}

// 通过minio重构部分
func (vr *VideoRepository) ListByAuthorID(ctx context.Context, authorID int64) ([]entity.Video, error) {

}
func (vr *VideoRepository) GetByID(ctx context.Context, id uint) (*entity.Video, error) {

}
