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

// GetAllAdminUsers gets all admin users
func (s *AdminUserService) GetAllAdminUsers(ctx context.Context) ([]models.AdminUser, error) {
	return s.repo.GetAll(ctx)
}

// UpdateAdminUser updates an admin user
func (s *AdminUserService) UpdateAdminUser(ctx context.Context, id uint, req *models.UpdateAdminUserRequest) (*models.AdminUser, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Username != "" {
		existing, err := s.repo.GetByUsername(ctx, req.Username)
		if err == nil && existing.ID != id {
			return nil, errors.New("username already exists")
		}
		user.Username = req.Username
	}

	if req.Password != "" {
		hashedPassword, err := HashPassword(req.Password)
		if err != nil {
			return nil, err
		}
		user.Password = hashedPassword
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
		return nil, err
	}

	return user, nil
}

// DeleteAdminUser deletes an admin user
func (s *AdminUserService) DeleteAdminUser(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
