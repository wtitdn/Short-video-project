package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/wtitdn/renew_video/internal/controller/apierror"
	"github.com/wtitdn/renew_video/internal/entity"
	"github.com/wtitdn/renew_video/internal/usecase"
	"github.com/wtitdn/renew_video/pkg/jwt"
)

type AdminHandler struct {
	adminService *usecase.AdminService
}

func NewAdminHandler(adminService *usecase.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (a *AdminHandler) Login(c *gin.Context) {
	var req entity.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	admin, err := a.adminService.Login(c.Request.Context(), req.AdminNameOrEmail, req.AdminPassword)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, entity.AdminLoginResponse{Token: admin.Token, RefreshToken: admin.RefreshToken, ID: admin.AdminID, AdminName: admin.AdminName})
}
func (a *AdminHandler) Logout(c *gin.Context) {
	adminID, err := jwt.GetAdminID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := a.adminService.Logout(c.Request.Context(), adminID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "account logged out"})
}

// 改密码前，邮箱发送验证码
func (a *AdminHandler) SendCode(c *gin.Context) {
	adminEmail, err := jwt.GetAdminEmail(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := a.adminService.SendCode(c.Request.Context(), adminEmail); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "code sent"})
}

func (a *AdminHandler) ChangePassword(c *gin.Context) {
	var req entity.AdminChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := a.adminService.ChangePassword(c.Request.Context(), req.AdminEmail, req.NewPassword, req.Code); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "password changed successfully"})
}
