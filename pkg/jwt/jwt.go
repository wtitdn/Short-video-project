package jwt

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wtitdn/renew_video/internal/middleware/auth"
	"github.com/wtitdn/renew_video/internal/repo"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
)

// JWTAuth check jwt token and ensure it matches the currently stored token.
func JWTAuth(accountRepo *repo.AccountRepository, cache *rediscache.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		tokenString := parts[1]

		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		//对比redis和db里面的token
		check(c, claims, tokenString, accountRepo, cache)
	}
}

// AdminJWTAuth check jwt token and ensure it matches the currently stored token.
func AdminJWTAuth(adminRepo *repo.AdminRepository, cache *rediscache.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		tokenString := parts[1]
		// token解析为用户信息
		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		adminID := claims.AccountID

		admin, err := adminRepo.FindByID(c.Request.Context(), adminID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin not found"})
			return
		}

		key := cache.Key("admin:%d", adminID)

		if cache != nil {
			cacheCtx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
			defer cancel()

			b, err := cache.GetBytes(cacheCtx, key)
			// 如果redis力有数据，比较
			if err == nil {
				if string(b) != tokenString {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
					return
				}
				// gin上下文更新
				
				c.Set("adminID", admin.AdminID)
				c.Set("email", admin.AdminEmail)
				c.Set("adminName", admin.AdminName)
				c.Next()
				return
			}
		}

		if admin.Token == "" || admin.Token != tokenString {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			return
		}

		if cache != nil {
			cacheCtx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
			defer cancel()

			if err := cache.SetBytes(cacheCtx, key, []byte(tokenString), 24*time.Hour); err != nil {
				log.Printf("failed to set admin cache: %v", err)
			}
		}
		// gin上下文更新
		c.Set("adminID", admin.AdminID)
		c.Set("email", admin.AdminEmail)
		c.Set("adminName", admin.AdminName)
		c.Next()
	}
}
func SoftJWTAuth(accountRepo *repo.AccountRepository, cache *rediscache.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}

		tokenString := parts[1]

		claims, err := auth.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		check(c, claims, tokenString, accountRepo, cache)
	}
}

func check(c *gin.Context, claims *auth.Claims, tokenString string, accountRepo *repo.AccountRepository, cache *rediscache.Client) {
	key := cache.Key("account:%d", claims.AccountID)

	// 先查 Redis
	if cache != nil {
		cacheCtx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
		defer cancel()

		b, err := cache.GetBytes(cacheCtx, key)
		if err == nil {
			if string(b) != tokenString {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
				return
			}
			c.Set("accountID", claims.AccountID)
			c.Set("username", claims.Username)
			c.Next()
			return
		}
	}

	// Redis 故障/未启用：查 DB 兜底
	accountInfo, err := accountRepo.FindByID(c.Request.Context(), claims.AccountID)
	if err != nil || accountInfo.Token == "" || accountInfo.Token != tokenString {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
		return
	}

	if cache != nil {
		cacheCtx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
		defer cancel()

		if err := cache.SetBytes(cacheCtx, key, []byte(tokenString), 24*time.Hour); err != nil {
			log.Printf("failed to set cache: %v", err)
		}
	}

	c.Set("accountID", claims.AccountID)
	c.Set("username", claims.Username)
	c.Next()

}

func GetAccountID(c *gin.Context) (uint, error) {
	uidValue, exists := c.Get("accountID")
	if !exists {
		return 0, errors.New("accountID not found")
	}

	accountID, ok := uidValue.(uint)
	if !ok {
		return 0, errors.New("accountID has invalid type")
	}

	return accountID, nil
}

func GetUsername(c *gin.Context) (string, error) {
	val, exists := c.Get("username")
	if !exists {
		return "", errors.New("username not found")
	}

	username, ok := val.(string)
	if !ok {
		return "", errors.New("username has invalid type")
	}

	return username, nil
}
func GetAdminID(c *gin.Context) (uint, error) {
	value, exists := c.Get("adminID")
	if !exists {
		return 0, errors.New("adminID not found")
	}

	adminID, ok := value.(uint)
	if !ok {
		return 0, errors.New("adminID has invalid type")
	}

	return adminID, nil
}

func GetAdminEmail(c *gin.Context) (string, error) {
	value, exists := c.Get("email")
	if !exists {
		return "", errors.New("email not found")
	}

	email, ok := value.(string)
	if !ok {
		return "", errors.New("email has invalid type")
	}

	return email, nil
}
