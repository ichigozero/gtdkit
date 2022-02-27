package gorm

import (
	"github.com/ichigozero/gtdkit/backend/usersvc"
	libgorm "gorm.io/gorm"
)

type userRepository struct {
	db *libgorm.DB
}

func NewUserRepository(db *libgorm.DB) usersvc.UserRepository {
	return &userRepository{db}
}

func (u *userRepository) GetUser(username string) *usersvc.User {
	var user usersvc.User
	u.db.Where("name = ?", username).First(&user)

	return &user
}

func (u *userRepository) IsExists(id uint64) (bool, error) {
	var user usersvc.User
	u.db.First(&user, id)

	if user.ID == 0 {
		return false, usersvc.ErrUserNotFound
	}

	return true, nil
}
