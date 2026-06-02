package repo

import (
	"context"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type AdminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (ar *AdminRepository) FindByEmail(ctx context.Context, email string) (*entity.Admin, error) {
	var admin entity.Admin
	if err := ar.db.WithContext(ctx).
		Where("admin_email = ?", email).
		First(&admin).
		Error; err != nil {
		return nil, err
	}
	return &admin, nil
}
func (ar *AdminRepository) FindByID(ctx context.Context, id uint) (*entity.Admin, error) {
	var admin entity.Admin
	if err := ar.db.WithContext(ctx).Where("admin_id = ?", id).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}
func (ar *AdminRepository) FindByUsername(ctx context.Context, username string) (*entity.Admin, error) {
	var admin entity.Admin
	if err := ar.db.WithContext(ctx).
		Where("admin_name = ?", username).
		First(&admin).
		Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

// update需要加上.model
func (ar *AdminRepository) ChangePassword(ctx context.Context, email string, password []byte) error {
	if err := ar.db.WithContext(ctx).Model(&entity.Admin{}).Where("admin_email = ?", email).Update("admin_password", string(password)).Error; err != nil {
		return err
	}
	return nil
}

func (ar *AdminRepository) Login(ctx context.Context, id uint, token, refreshToken string) error {
	if err := ar.db.WithContext(ctx).Model(&entity.Admin{}).Where(" admin_id = ?", id).Updates(map[string]interface{}{"token": token, "refresh_token": refreshToken}).Error; err != nil {
		return err
	}
	return nil
}
func (ar *AdminRepository) Logout(ctx context.Context, id uint) error {
	if err := ar.db.WithContext(ctx).Model(&entity.Admin{}).Where("admin_id = ?", id).Updates(map[string]interface{}{"token": "", "refresh_token": ""}).Error; err != nil {
		return err
	}
	return nil
}
