package usersvc

import "errors"

type User struct {
	ID       uint64 `json:"id" gorm:"primaryKey"`
	Name     string `json:"name" gorm:"unique"`
	Password string `json:"password"`
}

type UserRepository interface {
	GetUser(username string) *User
	IsExists(id uint64) (bool, error)
}

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUserNotFound    = errors.New("user not found")
)
