package usecase

import (
	"context"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/wtitdn/renew_video/internal/entity"
	"github.com/wtitdn/renew_video/internal/middleware/auth"
	"github.com/wtitdn/renew_video/internal/repo"
	smtp "github.com/wtitdn/renew_video/pkg/SMTP"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AccountService struct {
	accountRepository *repo.AccountRepository
	cache             *rediscache.Client
	minioRepo         *repo.MinioRepository
	smtpClient        *smtp.SmtpClient
	captchaClient     CaptchaService
}

type CaptchaService interface {
	GenerateIdAndImage(ip string) (id, b64s, ans string, err error)
	Verify(ip string, answer string) (bool, error)
}

var (
	ErrUsernameTaken       = errors.New("username already exists")
	ErrNewUsernameRequired = errors.New("new_username is required")
)

const (
	avatarBucket = "imagesys"
	avatarExpiry = 24 * time.Hour
)

// 需要测试captcha的话，修改captcha为测试文件
func NewAccountService(accountRepository *repo.AccountRepository, cache *rediscache.Client, minioRepo *repo.MinioRepository, smtpClient *smtp.SmtpClient, captchaClient CaptchaService) *AccountService {
	return &AccountService{accountRepository: accountRepository, cache: cache, minioRepo: minioRepo, smtpClient: smtpClient, captchaClient: captchaClient}
}
func (as *AccountService) SendEmailCode(ctx context.Context, mail string) error {
	if as.cache == nil {
		return errors.New("验证码服务不可用")
	}
	if as.smtpClient == nil {
		return errors.New("邮件服务不可用")
	}
	code := as.smtpClient.GenerateCode()
	if err := as.smtpClient.SendCode(mail, code); err != nil {
		return err
	}
	key := as.cache.Key("email:register:code:%s", mail)

	if err := as.cache.SetBytes(ctx, key, []byte(code), 5*time.Minute); err != nil {
		return err
	}

	return nil
}
func (as *AccountService) CreateAccount(ctx context.Context, account *entity.Account, verifyCode string) error {
	if as.cache == nil {
		return errors.New("验证码服务不可用")
	}
	key := as.cache.Key("email:register:code:%s", account.Email)
	codeBytes, err := as.cache.GetBytes(ctx, key)
	if err != nil {
		return errors.New("验证码不存在或已过期")
	}
	savedCode := strings.TrimSpace(string(codeBytes))
	inputCode := strings.TrimSpace(verifyCode)
	if savedCode != inputCode {
		return errors.New("验证码错误")
	}
	//密码哈希存入
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(account.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	account.Password = string(passwordHash)
	if err := as.accountRepository.CreateAccount(ctx, account); err != nil {
		return err
	}
	return nil
}

func (as *AccountService) Rename(ctx context.Context, accountID uint, newUsername string) (string, error) {
	if newUsername == "" {
		return "", ErrNewUsernameRequired
	}

	token, err := auth.GenerateToken(accountID, newUsername)
	if err != nil {
		return "", err
	}
	//调用Repo层的方法
	if err := as.accountRepository.RenameWithToken(ctx, accountID, newUsername, token); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return "", ErrUsernameTaken
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		return "", err
	}
	if as.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		if err := as.cache.SetBytes(cacheCtx, as.cache.Key("account:%d", accountID), []byte(token), 24*time.Hour); err != nil {
			log.Printf("failed to set cache: %v", err)
		}
	}
	return token, nil
}

func (as *AccountService) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	account, err := as.FindByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(oldPassword)); err != nil {
		return err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := as.accountRepository.ChangePassword(ctx, account.ID, string(passwordHash)); err != nil {
		return err
	}
	if err := as.Logout(ctx, account.ID); err != nil {
		return err
	}
	return nil
}

func (as *AccountService) FindByID(ctx context.Context, id uint) (*entity.Account, error) {
	if account, err := as.accountRepository.FindByID(ctx, id); err != nil {
		return nil, err
	} else {
		as.fillAvatarURL(ctx, account)
		return account, nil
	}
}

func (as *AccountService) FindByUsername(ctx context.Context, username string) (*entity.Account, error) {
	if account, err := as.accountRepository.FindByUsername(ctx, username); err != nil {
		return nil, err
	} else {
		as.fillAvatarURL(ctx, account)
		return account, nil
	}
}
func (as *AccountService) GetCaptcha(ctx context.Context) (b64s string, err error) {
	ip, _ := ctx.Value("client_ip").(string)
	_, b64s, _, err = as.captchaClient.GenerateIdAndImage(ip)
	if err != nil {
		return "", err
	}
	return b64s, nil
}

func (as *AccountService) VerifyCaptcha(ctx context.Context, captcha string) (bool, error) {
	ip, _ := ctx.Value("client_ip").(string)
	result, err := as.captchaClient.Verify(ip, captcha)
	if err != nil {
		return false, err
	}
	return result, nil
}
func (as *AccountService) Login(ctx context.Context, username, password string) (string, string, error) {

	account, err := as.FindByUsername(ctx, username)
	if err != nil {
		return "", "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password)); err != nil {
		return "", "", err
	}
	accessToken, err := auth.GenerateToken(account.ID, account.Username)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := auth.GenerateRefreshToken(account.ID)
	if err != nil {
		return "", "", err
	}
	//向db存入token
	if err := as.accountRepository.Login(ctx, account.ID, accessToken, refreshToken); err != nil {
		return "", "", err
	}
	//向redis存入token
	if as.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		if err := as.cache.SetBytes(cacheCtx, as.cache.Key("account:%d:token", account.ID), []byte(accessToken), 24*time.Hour); err != nil {
			log.Printf("failed to set cache: %v", err)
		}
		if err := as.cache.SetBytes(cacheCtx, as.cache.Key("account:%d:refreshToken", account.ID), []byte(refreshToken), 7*24*time.Hour); err != nil {
			log.Printf("failed to set refresh cache: %v", err)
		}
		//根据refreshtoken设置id 把refreshtoken变成key进行存储
		if err := as.cache.SetBytes(cacheCtx, as.cache.Key("refresh:%s", refreshToken), []byte(strconv.FormatUint(uint64(account.ID), 10)), 7*24*time.Hour); err != nil {
			log.Printf("failed to set refresh lookup: %v", err)
		}
	}
	return accessToken, refreshToken, nil
}

func (as *AccountService) Logout(ctx context.Context, accountID uint) error {
	account, err := as.FindByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Token == "" {
		return nil
	}
	if as.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		if err := as.cache.Del(cacheCtx, as.cache.Key("account:%d", account.ID)); err != nil {
			log.Printf("failed to del cache: %v", err)
		}
		if err := as.cache.Del(cacheCtx, as.cache.Key("account:%d:refresh", account.ID)); err != nil {
			log.Printf("failed to del refresh cache: %v", err)
		}
		if account.RefreshToken != "" {
			as.cache.Del(cacheCtx, as.cache.Key("refresh:%s", account.RefreshToken))
		}
	}
	return as.accountRepository.Logout(ctx, account.ID)
}
func (s *AccountService) UploadAvatar(ctx context.Context, accountID uint, objectKey, contentType string, reader io.Reader, size int64) (string, error) {
	if s.minioRepo == nil {
		return "", errors.New("minio repo is nil")
	}

	if err := s.minioRepo.EnsureBucket(ctx, avatarBucket); err != nil {
		return "", err
	}

	if err := s.minioRepo.UploadObject(ctx, avatarBucket, objectKey, contentType, reader, size); err != nil {
		return "", err
	}

	if err := s.UpdateAvatar(ctx, accountID, objectKey); err != nil {
		return "", err
	}

	avatarURL, err := s.minioRepo.PresignedGetURL(ctx, avatarBucket, objectKey, avatarExpiry)
	if err != nil {
		return "", err
	}

	return avatarURL, nil
}
func (as *AccountService) UpdateAvatar(ctx context.Context, accountID uint, avatarURL string) error {
	return as.accountRepository.UpdateAvatar(ctx, accountID, avatarURL)
}

func (as *AccountService) FindAll(ctx context.Context) ([]*entity.Account, error) {
	accounts, err := as.accountRepository.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		as.fillAvatarURL(ctx, account)
	}
	return accounts, nil
}

func (as *AccountService) UpdateProfile(ctx context.Context, accountID uint, req *entity.UpdateProfileRequest) error {
	updates := map[string]interface{}{}
	if req.Bio != "" {
		updates["bio"] = strings.TrimSpace(req.Bio)
	}
	if req.AvatarURL != "" {
		updates["avatar_url"] = strings.TrimSpace(req.AvatarURL)
	}
	if len(updates) == 0 {
		return errors.New("nothing to update")
	}
	return as.accountRepository.UpdateFields(ctx, accountID, updates)
}

func (as *AccountService) fillAvatarURL(ctx context.Context, account *entity.Account) {
	if account == nil || account.AvatarURL == "" || as.minioRepo == nil {
		return
	}
	if isExternalURL(account.AvatarURL) {
		return
	}
	avatarURL, err := as.minioRepo.PresignedGetURL(ctx, avatarBucket, account.AvatarURL, avatarExpiry)
	if err != nil {
		log.Printf("failed to presign avatar %q: %v", account.AvatarURL, err)
		return
	}
	account.AvatarURL = avatarURL
}

func isExternalURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "blob:")
}

func (as *AccountService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, uint, string, error) {
	if refreshToken == "" {
		return "", 0, "", errors.New("refresh token is empty")
	}
	if as.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		b, err := as.cache.GetBytes(cacheCtx, as.cache.Key("refresh:%s", refreshToken))
		if err == nil {
			idStr := string(b)
			id, parseErr := strconv.ParseUint(idStr, 10, 64)
			if parseErr == nil {
				account, err := as.FindByID(ctx, uint(id))
				if err == nil && account != nil && account.RefreshToken == refreshToken {
					newToken, err := auth.GenerateToken(account.ID, account.Username)
					if err != nil {
						return "", 0, "", err
					}
					as.accountRepository.UpdateToken(ctx, account.ID, newToken)
					as.cache.SetBytes(cacheCtx, as.cache.Key("account:%d", account.ID), []byte(newToken), 24*time.Hour)
					return newToken, account.ID, account.Username, nil
				}
			}
		}
	}
	accounts, err := as.FindAll(ctx)
	if err != nil {
		return "", 0, "", err
	}
	for _, acc := range accounts {
		if acc.RefreshToken == refreshToken {
			newToken, err := auth.GenerateToken(acc.ID, acc.Username)
			if err != nil {
				return "", 0, "", err
			}
			as.accountRepository.UpdateToken(ctx, acc.ID, newToken)
			return newToken, acc.ID, acc.Username, nil
		}
	}
	return "", 0, "", errors.New("invalid refresh token")
}
