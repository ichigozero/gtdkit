package gorm

import (
	"github.com/ichigozero/gtdkit/backend/usersvc"
	stdgorm "gorm.io/gorm"
)

type userRepository struct {
	db *stdgorm.DB
}

func NewUserRepository(db *stdgorm.DB) usersvc.UserRepository {
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
