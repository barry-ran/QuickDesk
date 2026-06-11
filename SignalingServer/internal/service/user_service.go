package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"quickdesk/signaling/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService owns the end-user account lifecycle: register, login
// credential check, profile updates, password changes, hard delete.
// Token issuance is the TokenService's job; this service only deals with
// PostgreSQL state.
type UserService struct {
	db         *gorm.DB
	deviceRepo interface {
		UnbindByUser(ctx context.Context, userID uint) error
	}
}

// UserServiceDeps lets us pass the device repo as an interface so tests
// can stub it without pulling in the full repository package.
type UserServiceDeps struct {
	DeviceUnbinder interface {
		UnbindByUser(ctx context.Context, userID uint) error
	}
}

func NewUserService(db *gorm.DB, deps UserServiceDeps) *UserService {
	return &UserService{db: db, deviceRepo: deps.DeviceUnbinder}
}

// Sentinel errors.
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUsernameTaken     = errors.New("username already taken")
	ErrPhoneTaken        = errors.New("phone already taken")
	ErrEmailTaken        = errors.New("email already taken")
	ErrBadCredentials    = errors.New("invalid credentials")
	ErrAccountDisabled   = errors.New("account is disabled")
	ErrPasswordWeak      = errors.New("password does not meet complexity requirements")
	ErrPhoneInvalid      = errors.New("phone number is not valid")
	ErrOldPasswordWrong  = errors.New("old password does not match")
)

// -----------------------------------------------------------------------
// Validation helpers
// -----------------------------------------------------------------------

// ValidatePassword enforces the policy from the old validatePassword helper:
// 鈮? chars, contains at least one letter and one digit. Kept verbatim to
// stay compatible with existing Qt/Web client expectations.
func ValidatePassword(p string) error {
	if len(p) < 8 {
		return ErrPasswordWeak
	}
	var hasLetter, hasDigit bool
	for _, r := range p {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return ErrPasswordWeak
	}
	return nil
}

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func ValidateEmail(s string) bool {
	return emailRegex.MatchString(strings.TrimSpace(s))
}

// -----------------------------------------------------------------------
// Create / read
// -----------------------------------------------------------------------

// RegisterInput is the normalised payload for POST /v1/auth/register.
type RegisterInput struct {
	Username string
	Password string
	Phone    string // optional
	Email    string // optional
}

func (s *UserService) Register(ctx context.Context, in RegisterInput) (*models.User, error) {
	if err := ValidatePassword(in.Password); err != nil {
		return nil, err
	}
	if in.Phone != "" && !ValidatePhone(in.Phone) {
		return nil, ErrPhoneInvalid
	}
	if in.Email != "" && !ValidateEmail(in.Email) {
		return nil, errors.New("email format invalid")
	}

	var existing models.User
	if err := s.db.WithContext(ctx).Where("username = ?", in.Username).First(&existing).Error; err == nil {
		return nil, ErrUsernameTaken
	}
	if in.Phone != "" {
		if err := s.db.WithContext(ctx).Where("phone = ?", in.Phone).First(&existing).Error; err == nil {
			return nil, ErrPhoneTaken
		}
	}
	if in.Email != "" {
		if err := s.db.WithContext(ctx).Where("email = ?", in.Email).First(&existing).Error; err == nil {
			return nil, ErrEmailTaken
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &models.User{
		Username:    in.Username,
		Phone:       in.Phone,
		Email:       in.Email,
		Password:    string(hash),
		Level:       "V1",
		ChannelType: "鍏ㄧ悆",
		Status:      true,
	}
	if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// LoginByIdentifier accepts a username / phone / email as `identifier` and
// verifies the password. Returns ErrBadCredentials on any mismatch so the
// server never leaks whether the account exists.
func (s *UserService) LoginByIdentifier(ctx context.Context, identifier, password string) (*models.User, error) {
	var u models.User
	err := s.db.WithContext(ctx).
		Where("username = ? OR phone = ? OR email = ?", identifier, identifier, identifier).
		First(&u).Error
	if err != nil {
		return nil, ErrBadCredentials
	}
	if !u.Status {
		return nil, ErrAccountDisabled
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) != nil {
		return nil, ErrBadCredentials
	}
	return &u, nil
}

// LoginByPhone is used for SMS-code login: the caller has already verified
// the code and now just needs the user row (if any).
func (s *UserService) LoginByPhone(ctx context.Context, phone string) (*models.User, error) {
	var u models.User
	err := s.db.WithContext(ctx).Where("phone = ?", phone).First(&u).Error
	if err != nil {
		return nil, ErrUserNotFound
	}
	if !u.Status {
		return nil, ErrAccountDisabled
	}
	return &u, nil
}

func (s *UserService) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	err := s.db.WithContext(ctx).First(&u, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *UserService) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	var u models.User
	err := s.db.WithContext(ctx).Where("phone = ?", phone).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// -----------------------------------------------------------------------
// Profile updates (PUT /v1/me/*)
// -----------------------------------------------------------------------

func (s *UserService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	u, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(oldPassword)) != nil {
		return ErrOldPasswordWrong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("password", string(hash)).Error
}

// ResetPasswordByPhone is used by the SMS password-reset flow; caller has
// already verified the code.
func (s *UserService) ResetPasswordByPhone(ctx context.Context, phone, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Model(&models.User{}).
		Where("phone = ?", phone).
		Update("password", string(hash))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *UserService) ChangeUsername(ctx context.Context, userID uint, newUsername string) error {
	newUsername = strings.TrimSpace(newUsername)
	if newUsername == "" {
		return errors.New("username is required")
	}
	var existing models.User
	if err := s.db.WithContext(ctx).
		Where("username = ? AND id != ?", newUsername, userID).
		First(&existing).Error; err == nil {
		return ErrUsernameTaken
	}
	return s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("username", newUsername).Error
}

func (s *UserService) ChangePhone(ctx context.Context, userID uint, newPhone string) error {
	newPhone = strings.TrimSpace(newPhone)
	if newPhone != "" && !ValidatePhone(newPhone) {
		return ErrPhoneInvalid
	}
	if newPhone != "" {
		var existing models.User
		if err := s.db.WithContext(ctx).
			Where("phone = ? AND id != ?", newPhone, userID).
			First(&existing).Error; err == nil {
			return ErrPhoneTaken
		}
	}
	return s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("phone", newPhone).Error
}

func (s *UserService) ChangeEmail(ctx context.Context, userID uint, newEmail string) error {
	newEmail = strings.TrimSpace(newEmail)
	if newEmail != "" && !ValidateEmail(newEmail) {
		return errors.New("email format invalid")
	}
	if newEmail != "" {
		var existing models.User
		if err := s.db.WithContext(ctx).
			Where("email = ? AND id != ?", newEmail, userID).
			First(&existing).Error; err == nil {
			return ErrEmailTaken
		}
	}
	return s.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("email", newEmail).Error
}

// -----------------------------------------------------------------------
// Hard delete (admin action)
// -----------------------------------------------------------------------

// Delete hard-removes a user row and cascades dependent data. Explicitly
// unbinds the user from devices (clearing user_id + logged_in) per
// 搂2.4 bug-闃叉姢 鈥?otherwise a crashed-but-still-online host would keep
// `logged_in=true` pointing at a deleted user.
func (s *UserService) Delete(ctx context.Context, userID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserFavorite{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserDevice{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.ConnectionHistory{}).Error; err != nil {
			return err
		}
		// Clear FK + intent on any devices this user owned.
		if err := tx.Model(&models.Device{}).
			Where("user_id = ?", userID).
			Updates(map[string]interface{}{
				"user_id":          nil,
				"logged_in": false,
			}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.User{}, userID).Error
	})
}

// -----------------------------------------------------------------------
// Admin-only helpers used by /v1/admin/users
// -----------------------------------------------------------------------

// UserAdminListParams carries the cursor-keyset filters used by the admin
// web. `AfterID` supports keyset-style "WHERE id < after" pagination with
// `Limit+1` lookahead to compute next_cursor.
type UserAdminListParams struct {
	AfterID     uint
	Limit       int
	Sort, Order string
	Search      string
	Level       string
	Status      *bool
	ChannelType string
}

func (s *UserService) AdminList(ctx context.Context, p UserAdminListParams) ([]models.User, int64, error) {
	q := s.db.WithContext(ctx).Model(&models.User{})
	if p.Search != "" {
		like := "%" + p.Search + "%"
		q = q.Where("username LIKE ? OR phone LIKE ? OR email LIKE ?", like, like, like)
	}
	if p.Level != "" {
		q = q.Where("level = ?", p.Level)
	}
	if p.Status != nil {
		q = q.Where("status = ?", *p.Status)
	}
	if p.ChannelType != "" {
		q = q.Where("channel_type = ?", p.ChannelType)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Keyset pagination on id DESC; `AfterID == 0` means first page.
	if p.AfterID > 0 {
		q = q.Where("id < ?", p.AfterID)
	}
	sort := p.Sort
	if sort == "" {
		sort = "id"
	}
	order := p.Order
	if order == "" {
		order = "desc"
	}

	var users []models.User
	err := q.Order(sort + " " + order).Limit(p.Limit).Find(&users).Error
	return users, total, err
}

func (s *UserService) AdminUpdate(ctx context.Context, userID uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	res := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *UserService) CountSince(ctx context.Context, since time.Time) (int64, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&models.User{}).Where("created_at >= ?", since).Count(&n).Error
	return n, err
}
