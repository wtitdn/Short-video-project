package usecase

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/wtitdn/renew_video/internal/entity"
	"github.com/wtitdn/renew_video/internal/middleware/auth"
	"github.com/wtitdn/renew_video/internal/repo"
	smtp "github.com/wtitdn/renew_video/pkg/SMTP"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
	"golang.org/x/crypto/bcrypt"
)

type AdminService struct {
	adminRepository *repo.AdminRepository
	cache           *rediscache.Client
	minioRepo       *repo.MinioRepository
	smtp            *smtp.SmtpClient
}

func NewAdminService(adminRepository *repo.AdminRepository, cache *rediscache.Client, minioRepo *repo.MinioRepository, smtp *smtp.SmtpClient) *AdminService {
	return &AdminService{adminRepository, cache, minioRepo, smtp}
}

func (as *AdminService) Login(ctx context.Context, adminNameOrEmail, password string) (*entity.Admin, error) {
	admin, err := as.adminRepository.FindByUsername(ctx, adminNameOrEmail)
	if err != nil {
		admin, err = as.adminRepository.FindByEmail(ctx, adminNameOrEmail)
		if err != nil {
			return nil, err
		}
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(admin.AdminPassword),
		[]byte(password),
	); err != nil {
		return nil, err
	}
	accessToken, err := auth.GenerateToken(admin.AdminID, admin.AdminName)
	if err != nil {
		return nil, err
	}
	refreshToken, err := auth.GenerateRefreshToken(admin.AdminID)
	if err != nil {
		return nil, err
	}
	//向db存入token
	if err := as.adminRepository.Login(ctx, admin.AdminID, accessToken, refreshToken); err != nil {
		return nil, err
	}
	//向redis存入token
	if as.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		if err := as.cache.SetBytes(cacheCtx, as.cache.Key("admin:%d", admin.AdminID), []byte(accessToken), 24*time.Hour); err != nil {
			log.Printf("failed to set cache: %v", err)
		}
		if err := as.cache.SetBytes(cacheCtx, as.cache.Key("admin:%d:refresh", admin.AdminID), []byte(refreshToken), 7*24*time.Hour); err != nil {
			log.Printf("failed to set refresh cache: %v", err)
		}
		if err := as.cache.SetBytes(cacheCtx, as.cache.Key("admin:%s", refreshToken), []byte(strconv.FormatUint(uint64(admin.AdminID), 10)), 7*24*time.Hour); err != nil {
			log.Printf("failed to set refresh lookup: %v", err)
		}
	}
	admin.Token = accessToken
	admin.RefreshToken = refreshToken
	return admin, nil
}
func (as *AdminService) Logout(ctx context.Context, adminID uint) error {
	admin, err := as.adminRepository.FindByID(ctx, adminID)
	if err != nil {
		return err
	}
	if admin.Token == "" {
		return nil
	}
	if as.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		if err := as.cache.Del(cacheCtx, as.cache.Key("admin:%d", admin.AdminID)); err != nil {
			log.Printf("failed to set cache: %v", err)
		}
		if err := as.cache.Del(cacheCtx, as.cache.Key("admin:%d:refresh", admin.AdminID)); err != nil {
			log.Printf("failed to set refresh cache: %v", err)
		}
		if admin.RefreshToken != "" {
			as.cache.Del(cacheCtx, as.cache.Key("refresh:%s", admin.RefreshToken))
		}
	}
	return as.adminRepository.Logout(ctx, admin.AdminID)
}
func (as *AdminService) SendCode(ctx context.Context, email string) error {
	if as.cache == nil {
		return errors.New("验证码服务不可用")
	}
	if as.smtp == nil {
		return errors.New("邮件服务不可用")
	}
	code := as.smtp.GenerateCode()
	if err := as.smtp.SendCode(email, code); err != nil {
		return err
	}
	key := as.cache.Key("admin:change-password:code:%s", email)

	if err := as.cache.SetBytes(ctx, key, []byte(code), 5*time.Minute); err != nil {
		return err
	}

	return nil
}
func (as *AdminService) ChangePassword(ctx context.Context, AdminEmail, newPassword, code string) error {
	if as.cache == nil {
		return errors.New("验证码服务不可用")
	}
	key := as.cache.Key("admin:change-password:code:%s", AdminEmail)
	codeBytes, err := as.cache.GetBytes(ctx, key)
	if err != nil {
		return errors.New("验证码不存在或已过期")
	}
	savedCode := strings.TrimSpace(string(codeBytes))
	inputCode := strings.TrimSpace(code)
	if savedCode != inputCode {
		return errors.New("验证码错误")
	}

	admin, err := as.adminRepository.FindByEmail(ctx, AdminEmail)
	if err != nil {
		return err
	}

	password, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err

	}
	if err := as.adminRepository.ChangePassword(ctx, admin.AdminEmail, password); err != nil {
		return err
	}
	_ = as.cache.Del(ctx, key)
	_ = as.adminRepository.Logout(ctx, admin.AdminID)
	return nil
}
