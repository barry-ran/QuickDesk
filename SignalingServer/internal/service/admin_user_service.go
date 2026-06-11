package service

import (
	"context"
	"errors"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// AdminUserService handles business logic for admin users
type AdminUserService struct {
	repo *repository.AdminUserRepository
}

// NewAdminUserService creates a new AdminUserService
func NewAdminUserService(repo *repository.AdminUserRepository) *AdminUserService {
	return &AdminUserService{repo: repo}
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// checkPasswordHash checks if a password matches a hash
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidateCredentials validates admin credentials and returns the user if valid
func (s *AdminUserService) ValidateCredentials(ctx context.Context, username, password string) (*models.AdminUser, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.Status {
		return nil, errors.New("account is disabled")
	}

	if !checkPasswordHash(password, user.Password) {
		return nil, errors.New("invalid credentials")
	}

	// Update last login time
	s.repo.UpdateLastLogin(ctx, user.ID)

	return user, nil
}

// CreateAdminUser creates a new admin user
func (s *AdminUserService) CreateAdminUser(ctx context.Context, req *models.CreateAdminUserRequest) (*models.AdminUser, error) {
	// Check if username already exists
	_, err := s.repo.GetByUsername(ctx, req.Username)
	if err == nil {
		return nil, errors.New("username already exists")
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	role := req.Role
	if role == "" {
		role = "admin"
	}

	user := &models.AdminUser{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		Role:     role,
		Status:   true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetAdminUserByID gets an admin user by ID
func (s *AdminUserService) GetAdminUserByID(ctx context.Context, id uint) (*models.AdminUser, error) {
	return s.repo.GetByID(ctx, id)
}

// GetAdminUserByUsername gets an admin user by username
func (s *AdminUserService) GetAdminUserByUsername(ctx context.Context, username string) (*models.AdminUser, error) {
	return s.repo.GetByUsername(ctx, username)
}

// GetAllAdminUsers gets all admin users
func (s *AdminUserService) GetAllAdminUsers(ctx context.Context) ([]models.AdminUser, error) {
	return s.repo.GetAll(ctx)
}

// UpdateAdminUser updates an admin user. The boolean second return tells
// the caller whether the password actually changed — admin handlers use
// it to revoke the target's active session families per §2.16
// "Admin session 改密码后 revoke 本账号全部 session（包括自己）".
func (s *AdminUserService) UpdateAdminUser(ctx context.Context, id uint, req *models.UpdateAdminUserRequest) (*models.AdminUser, bool, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, false, err
	}

	if req.Username != "" {
		existing, err := s.repo.GetByUsername(ctx, req.Username)
		if err == nil && existing.ID != id {
			return nil, false, errors.New("username already exists")
		}
		user.Username = req.Username
	}

	passwordChanged := false
	if req.Password != "" {
		hashedPassword, err := HashPassword(req.Password)
		if err != nil {
			return nil, false, err
		}
		user.Password = hashedPassword
		passwordChanged = true
	}

	if req.Email != "" {
		user.Email = req.Email
	}

	if req.Role != "" {
		user.Role = req.Role
	}

	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, false, err
	}

	return user, passwordChanged, nil
}

// DeleteAdminUser deletes an admin user
func (s *AdminUserService) DeleteAdminUser(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
