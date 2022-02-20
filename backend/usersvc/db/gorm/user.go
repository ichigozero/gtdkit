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

func (u *userRepository) UserID(username, password string) (uint64, error) {
	var user usersvc.User
	u.db.Select("id").Where("name = ? AND password = ?", username, password).Find(&user)

	if user.ID == 0 {
		return user.ID, usersvc.ErrUserNotFound
	}

	return user.ID, nil
}

func (u *userRepository) IsExists(id uint64) (bool, error) {
	var user usersvc.User
	u.db.First(&user, id)

	if user.ID == 0 {
		return false, usersvc.ErrUserNotFound
	}

	return true, nil
}
