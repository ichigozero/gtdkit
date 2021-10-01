package tasksvc

import "errors"

type Task struct {
	ID          uint64
	Title       string
	Description string
	Done        bool
	UserID      uint64
}

type TaskRepository interface {
	Create(title, description string, userID uint64) (Task, error)
	FindAll(userID uint64) ([]Task, error)
	Find(userID, taskID uint64) (Task, error)
	Update(task Task) (Task, error)
	Delete(userID, taskID uint64) (bool, error)
}

type Auth struct {
	AccessUUID string
	UserID     uint64
}

var (
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrUserIDContextMissing = errors.New("user ID was not passed through the context")
	ErrClaimsMissing        = errors.New("JWT claims was not passed through the context")
	ErrClaimsInvalid        = errors.New("JWT claims was invalid")
)
