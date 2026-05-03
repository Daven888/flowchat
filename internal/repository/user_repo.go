package repository

import (
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/pkg/mysql"
)

// UserRepository provides database access for user operations.
type UserRepository struct{}

// NewUserRepository creates a new UserRepository.
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// Create inserts a new user record.
func (r *UserRepository) Create(user *model.User) error {
	return mysql.DB.Create(user).Error
}

// FindByEmail looks up a user by email.
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	if err := mysql.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername looks up a user by username.
func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	if err := mysql.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID looks up a user by primary key.
func (r *UserRepository) FindByID(id int64) (*model.User, error) {
	var user model.User
	if err := mysql.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
