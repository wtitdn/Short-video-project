package db

import (
	"fmt"
	"strings"

	"github.com/wtitdn/renew_video/internal/config"
	"github.com/wtitdn/renew_video/internal/entity"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 表初始化
func NewDB(dbcfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbcfg.User, dbcfg.Password, dbcfg.Host, dbcfg.Port, dbcfg.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB, adminCfg config.AdminConfig) error {
	if err := db.AutoMigrate(
		&entity.Account{}, &entity.Video{}, &entity.Like{}, &entity.Comment{},
		&entity.Social{}, &entity.OutboxMsg{}, &entity.Tag{}, &entity.VideoTag{},
		&entity.Message{}, &entity.Notification{}, &entity.Admin{},
	); err != nil {
		return err
	}
	return seedAdmin(db, adminCfg)
}
func seedAdmin(db *gorm.DB, adminCfg config.AdminConfig) error {
	adminName := strings.TrimSpace(adminCfg.Name)
	adminPassword := strings.TrimSpace(adminCfg.Password)
	adminEmail := strings.TrimSpace(adminCfg.Email)
	if adminName == "" || adminPassword == "" || adminEmail == "" {
		return nil
	}

	var count int64
	if err := db.Model(&entity.Admin{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := entity.Admin{
		AdminName:     adminName,
		AdminPassword: string(passwordHash),
		AdminEmail:    adminEmail,
	}

	return db.Create(&admin).Error
}

func CloseDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
