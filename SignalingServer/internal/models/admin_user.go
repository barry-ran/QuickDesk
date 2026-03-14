package models

import (
	"time"
)

// AdminUser represents an administrator account
type AdminUser struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"` // Password hash, never expose in JSON
	Email     string    `json:"email" gorm:"index"`
	Role      string    `json:"role" gorm:"default:'admin'"` // admin, super_admin
	Status    bool      `json:"status" gorm:"default:true"`  // true: active, false: disabled
	LastLogin time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for AdminUser
func (AdminUser) TableName() string {
	return "admin_users"
}

// AdminUserResponse is the response format for admin user (without sensitive data)
type AdminUserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    bool      `json:"status"`
	LastLogin time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts AdminUser to AdminUserResponse
func (u *AdminUser) ToResponse() AdminUserResponse {
	return AdminUserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Status:    u.Status,
		LastLogin: u.LastLogin,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// CreateAdminUserRequest represents the request to create a new admin user
type CreateAdminUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email"`
	Role     string `json:"role" binding:"omitempty,oneof=admin super_admin"`
}

// UpdateAdminUserRequest represents the request to update an admin user
type UpdateAdminUserRequest struct {
	Username string `json:"username" binding:"omitempty,min=3,max=50"`
	Password string `json:"password" binding:"omitempty,min=6"`
	Email    string `json:"email"`
	Role     string `json:"role" binding:"omitempty,oneof=admin super_admin"`
	Status   *bool  `json:"status"`
}
