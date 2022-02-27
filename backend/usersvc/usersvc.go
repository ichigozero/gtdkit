package usersvc

import "errors"

type User struct {
	ID       uint64 `gorm:"primaryKey"`
	Name     string `gorm:"unique"`
	Password string
}

type UserRepository interface {
	GetUser(username string) *User
	IsExists(id uint64) (bool, error)
}

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUserNotFound    = errors.New("user not found")
)
