package repository

import (
	"context"
	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// AdminUserRepository handles database operations for admin users
type AdminUserRepository struct {
	db *gorm.DB
}

// NewAdminUserRepository creates a new AdminUserRepository
func NewAdminUserRepository(db *gorm.DB) *AdminUserRepository {
	return &AdminUserRepository{db: db}
}

// Create creates a new admin user
func (r *AdminUserRepository) Create(ctx context.Context, user *models.AdminUser) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID gets an admin user by ID
func (r *AdminUserRepository) GetByID(ctx context.Context, id uint) (*models.AdminUser, error) {
	var user models.AdminUser
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername gets an admin user by username
func (r *AdminUserRepository) GetByUsername(ctx context.Context, username string) (*models.AdminUser, error) {
	var user models.AdminUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAll gets all admin users
func (r *AdminUserRepository) GetAll(ctx context.Context) ([]models.AdminUser, error) {
	var users []models.AdminUser
	err := r.db.WithContext(ctx).Find(&users).Error
	return users, err
}

// Update updates an admin user
func (r *AdminUserRepository) Update(ctx context.Context, user *models.AdminUser) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete deletes an admin user
func (r *AdminUserRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.AdminUser{}, id).Error
}

// UpdateLastLogin updates the last login time
func (r *AdminUserRepository) UpdateLastLogin(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.AdminUser{}).Where("id = ?", id).Update("last_login", gorm.Expr("NOW()")).Error
}
