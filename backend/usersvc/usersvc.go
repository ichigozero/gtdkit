package usersvc

import "errors"

type User struct {
	ID       uint64
	Name     string
	Password string
}

type UserRepository interface {
	UserID(username, password string) (uint64, error)
}

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUserNotFound    = errors.New("user not found")
)
