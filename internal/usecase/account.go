package usecase

import (
	"context"
	"errors"

	"github.com/wtitdn/renew_video/internal/entity"
	"github.com/wtitdn/renew_video/internal/repo"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
)

type AccountService struct {
	accountRepository *repo.AccountRepository
	cache             *rediscache.Client
}

var (
	ErrUsernameTaken       = errors.New("username already exists")
	ErrNewUsernameRequired = errors.New("new_username is required")
)

func NewAccountService(accountRepository *repo.AccountRepository, cache *rediscache.Client) *AccountService {

}
func (as *AccountService) CreateAccount(ctx context.Context, account *entity.Account) error {

}
func (as *AccountService) Rename(ctx context.Context, accountID uint, newUsername string) (string, error) {

}
func (as *AccountService) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {

}

func (as *AccountService) DeleteAccount(ctx context.Context, accountID uint) error {

}
func (as *AccountService) FindByID(ctx context.Context, id uint) (*entity.Account, error) {

}
func (as *AccountService) FindByUsername(ctx context.Context, username string) (*entity.Account, error) {

}
func (as *AccountService) Login(ctx context.Context, username, password string) (string, string, error) {

}
func (as *AccountService) Logout(ctx context.Context, accountID uint) error {

}
func (as *AccountService) UpdateAvatar(ctx context.Context, accountID uint, avatarURL string) error {

}
func (as *AccountService) FindAll(ctx context.Context) ([]*entity.Account, error) {

}
func (as *AccountService) UpdateProfile(ctx context.Context, accountID uint, req *entity.UpdateProfileRequest) error {

}
func (as *AccountService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, uint, string, error) {
	
}
