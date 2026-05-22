package repo

import (
	"context"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type SocialRepository struct {
	db *gorm.DB
}

func NewSocialRepository(db *gorm.DB) *SocialRepository {
	return &SocialRepository{db: db}
}

func (r *SocialRepository) Follow(ctx context.Context, social *entity.Social) error {
	return r.db.WithContext(ctx).Create(social).Error
}

func (r *SocialRepository) Unfollow(ctx context.Context, social *entity.Social) error {
	return r.db.WithContext(ctx).
		Where("follower_id = ? AND vlogger_id = ?", social.FollowerID, social.VloggerID).
		Delete(&entity.Social{}).Error
}

func (r *SocialRepository) GetAllFollowers(ctx context.Context, VloggerID uint) ([]*entity.Account, error) {
	var relations []entity.Social
	if err := r.db.WithContext(ctx).
		Model(&entity.Social{}).
		Where("vlogger_id = ?", VloggerID).
		Limit(200).
		Find(&relations).Error; err != nil {
		return nil, err
	}

	followerIDs := make([]uint, 0, len(relations))
	for _, rel := range relations {
		followerIDs = append(followerIDs, rel.FollowerID)
	}
	if len(followerIDs) == 0 {
		return []*entity.Account{}, nil
	}

	var followers []*entity.Account
	if err := r.db.WithContext(ctx).
		Model(&entity.Account{}).
		Where("id IN ?", followerIDs).
		Find(&followers).Error; err != nil {
		return nil, err
	}
	return followers, nil
}

func (r *SocialRepository) GetAllVloggers(ctx context.Context, FollowerID uint) ([]*entity.Account, error) {
	var relations []entity.Social
	if err := r.db.WithContext(ctx).
		Model(&entity.Social{}).
		Where("follower_id = ?", FollowerID).
		Limit(200).
		Find(&relations).Error; err != nil {
		return nil, err
	}

	vloggerIDs := make([]uint, 0, len(relations))
	for _, rel := range relations {
		vloggerIDs = append(vloggerIDs, rel.VloggerID)
	}
	if len(vloggerIDs) == 0 {
		return []*entity.Account{}, nil
	}

	var vloggers []*entity.Account
	if err := r.db.WithContext(ctx).
		Model(&entity.Account{}).
		Where("id IN ?", vloggerIDs).
		Find(&vloggers).Error; err != nil {
		return nil, err
	}
	return vloggers, nil
}

func (r *SocialRepository) IsFollowed(ctx context.Context, social *entity.Social) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&entity.Social{}).
		Where("follower_id = ? AND vlogger_id = ?", social.FollowerID, social.VloggerID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *SocialRepository) CountFollowers(ctx context.Context, vloggerID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.Social{}).Where("vlogger_id = ?", vloggerID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *SocialRepository) CountVloggers(ctx context.Context, followerID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.Social{}).Where("follower_id = ?", followerID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
