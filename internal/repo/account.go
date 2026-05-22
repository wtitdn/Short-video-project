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
func (ar *AccountRepository) CreateAccountRepository(ctx context.Context, account *entity.Account) *AccountRepository {

}
func (ar *AccountRepository) DeleteAccountRepository(ctx context.Context, id uint) *AccountRepository {

}
func (ar *AccountRepository) Rename(ctx context.Context, id uint, s string) error {
}
func (ar *AccountRepository) ChangePassword(ctx context.Context, id uint, password string) error {

}
func (ar *AccountRepository) FindByID(ctx context.Context, id uint) (*entity.Account, error) {

}
func (ar *AccountRepository) FindByUserName(ctx context.Context, username string) (*entity.Account, error) {

}

// 登出时，要清除所有的token
func (ar *AccountRepository) Login(ctx context.Context, id uint, token, refreshToken string) error {

}

func (ar *AccountRepository) Logout(ctx context.Context, id uint) error {

}

// 更新头像 感觉可以用minio继续优化
func (ar *AccountRepository) UpdateAvatar(ctx context.Context, accountID uint, avatarURL string) error {

}

func (ar *AccountRepository) UpdateToken(ctx context.Context, id uint, token string) error {

}
func (ar *AccountRepository) UpdateFiles(ctx context.Context, id uint, updates map[string]interface{}) error {

}

// refresh token 兜底逻辑
func (ar *AccountRepository) FindAll(ctx context.Context) ([]*entity.Account, error) {

}
