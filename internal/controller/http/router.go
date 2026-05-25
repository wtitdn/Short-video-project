package http

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wtitdn/renew_video/internal/controller/http/handler"
	"github.com/wtitdn/renew_video/internal/repo"
	"github.com/wtitdn/renew_video/internal/usecase"
	"github.com/wtitdn/renew_video/pkg/jwt"
	storage "github.com/wtitdn/renew_video/pkg/minio"
	"github.com/wtitdn/renew_video/pkg/rabbitmq"
	"github.com/wtitdn/renew_video/pkg/ratelimit"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
	"gorm.io/gorm"
)

// 注册所有组件
func SetRouter(db *gorm.DB, cache *rediscache.Client, rmq *rabbitmq.RabbitMQ, mio *storage.Minio) *gin.Engine {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Printf("SetTrustedProxies failed: %v", err)
	}
	r.Static("/static", "./.run/uploads")
	//minio
	minioRepository := repo.NewMinioRepository(mio)
	//防抖中间件
	loginLimiter := ratelimit.Limit(cache, "account_login", 10, time.Minute, ratelimit.KeyByIP)
	registerLimiter := ratelimit.Limit(cache, "account_register", 5, time.Hour, ratelimit.KeyByIP)
	//account
	accountRepository := repo.NewAccountRepository(db)

	accountService := usecase.NewAccountService(accountRepository, cache, minioRepository)
	accountHandler := handler.NewAccountHandler(accountService)
	accountGroup := r.Group("/account")
	{
		accountGroup.POST("/register", registerLimiter, accountHandler.CreateAccount)
		accountGroup.POST("/login", loginLimiter, accountHandler.Login)
		accountGroup.POST("/changePassword", accountHandler.ChangePassword)
		accountGroup.POST("/findByID", accountHandler.FindByID)
		accountGroup.POST("/findByUsername", accountHandler.FindByUsername)
		accountGroup.POST("/refresh", accountHandler.Refresh)
	}
	protectedAccountGroup := accountGroup.Group("")
	protectedAccountGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		protectedAccountGroup.POST("/logout", accountHandler.Logout)
		protectedAccountGroup.POST("/rename", accountHandler.Rename)
		protectedAccountGroup.POST("/uploadAvatar", accountHandler.UploadAvatar)
		protectedAccountGroup.POST("/updateProfile", accountHandler.UpdateProfile)
	}

	return r
}
