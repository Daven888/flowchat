package service

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Daven888/flowchat/internal/auth"
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/internal/repository"
)

var (
	ErrUsernameTaken   = errors.New("username already exists")
	ErrEmailTaken      = errors.New("email already exists")
	ErrInvalidPassword = errors.New("invalid email or password")
	ErrUserNotFound    = errors.New("user not found")
)

// UserService handles user authentication business logic.
type UserService struct {
	repo      *repository.UserRepository
	jwtConfig auth.Config
}

// NewUserService creates a new UserService.
func NewUserService(repo *repository.UserRepository, jwtConfig auth.Config) *UserService {
	return &UserService{repo: repo, jwtConfig: jwtConfig}
}

// Register creates a new user account with bcrypt-hashed password.
func (s *UserService) Register(username, email, password string) (*model.User, error) {
	// Check username uniqueness
	if _, err := s.repo.FindByUsername(username); err == nil {
		return nil, ErrUsernameTaken
	}

	// Check email uniqueness
	if _, err := s.repo.FindByEmail(email); err == nil {
		return nil, ErrEmailTaken
	}

	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Status:       model.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login authenticates a user by email and password, returns a JWT token.
func (s *UserService) Login(email, password string) (string, *model.User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrInvalidPassword
		}
		return "", nil, err
	}

	// Reject disabled users
	if user.Status != model.UserStatusActive {
		return "", nil, ErrInvalidPassword
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidPassword
	}

	// Generate JWT token
	token, err := auth.GenerateToken(user.ID, s.jwtConfig)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

// GetProfile retrieves user information by ID.
func (s *UserService) GetProfile(userID int64) (*model.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
