package repo

import (
	"context"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

// 简单依赖注入
func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// 创建账户
func (ar *AccountRepository) CreateAccount(ctx context.Context, account *entity.Account) error {
	return ar.db.WithContext(ctx).Create(account).Error
}

// 删除账户
func (ar *AccountRepository) DeleteAccount(ctx context.Context, id uint) error {
	return ar.db.WithContext(ctx).Delete(&entity.Account{}, id).Error
}

// 重命名账户
func (ar *AccountRepository) Rename(ctx context.Context, id uint, s string) error {
	return ar.db.WithContext(ctx).Model(&entity.Account{}).Where("id = ?", id).Update("username", s).Error
}

// 修改密码
func (ar *AccountRepository) ChangePassword(ctx context.Context, id uint, password string) error {
	return ar.db.WithContext(ctx).Model(&entity.Account{}).Where("id = ?", id).Update("password", password).Error
}

func (ar *AccountRepository) FindByID(ctx context.Context, id uint) (*entity.Account, error) {
	var account entity.Account
	if err := ar.db.WithContext(ctx).First(&account, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// 根据用户名查找账户
func (ar *AccountRepository) FindByUserName(ctx context.Context, username string) (*entity.Account, error) {
	var account entity.Account
	if err := ar.db.WithContext(ctx).Where("username = ?", username).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// 登出时，要清除所有的token
func (ar *AccountRepository) Logout(ctx context.Context, id uint) error {
	return ar.db.WithContext(ctx).Model(&entity.Account{}).Where("id = ?", id).Updates(map[string]interface{}{
		"token":         "",
		"refresh_token": "",
	}).Error
}

// 登录时更新token
func (ar *AccountRepository) Login(ctx context.Context, id uint, token, refreshToken string) error {
	return ar.db.WithContext(ctx).Model(&entity.Account{}).Where("id = ?", id).Updates(map[string]interface{}{
		"token":         token,
		"refresh_token": refreshToken,
	}).Error
}

// 更新头像 感觉可以用minio继续优化
func (ar *AccountRepository) UpdateAvatar(ctx context.Context, accountID uint, avatarURL string) error {
	return ar.db.WithContext(ctx).Model(&entity.Account{}).Where("id = ?", accountID).Update("avatar", avatarURL).Error
}

// 更新token
func (ar *AccountRepository) UpdateToken(ctx context.Context, id uint, token string) error {
	return ar.db.WithContext(ctx).Model(&entity.Account{}).Where("id = ?", id).Update("token", token).Error
}

// 通用更新字段
func (ar *AccountRepository) UpdateFiles(ctx context.Context, id uint, updates map[string]interface{}) error {
	return ar.db.WithContext(ctx).Model(&entity.Account{}).Where("id = ?", id).Updates(updates).Error
}

// refresh token 兜底逻辑 - 获取所有账户
func (ar *AccountRepository) FindAll(ctx context.Context) ([]*entity.Account, error) {
	var accounts []*entity.Account
	if err := ar.db.WithContext(ctx).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}
