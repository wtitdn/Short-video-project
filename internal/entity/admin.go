package entity

type Admin struct {
	AdminID       uint   `gorm:"primaryKey" json:"id"`
	AdminName     string `json:"admin" binding:"required"`
	AdminPassword string `json:"admin_password" binding:"required"`
	AdminEmail    string `json:"admin_email" binding:"required,email"`
	Token         string `json:"-"`
	RefreshToken  string `json:"-"`
}

// DTO
type AdminLoginRequest struct {
	AdminNameOrEmail string `json:"admin_name" binding:"required"`
	AdminPassword    string `json:"admin_password" binding:"required"`
}
type AdminChangePasswordRequest struct {
	AdminEmail  string `json:"admin_email" binding:"required,email"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
	Code        string `json:"code" binding:"required"`
}
type AdminLoginResponse struct {
	ID           uint   `json:"id"`
	AdminName    string `json:"admin_name"`
	AdminEmail   string `json:"admin_email"`
	Token        string `json:"token"`
	RefreshToken string `json:"-"`
}
type AdminChangePasswordResponse struct {
}
