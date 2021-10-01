package gorm

import (
	"github.com/ichigozero/gtdkit/backend/tasksvc"
	stdgorm "gorm.io/gorm"
)

type taskRepository struct {
	db *stdgorm.DB
}

func NewTaskRepository(db *stdgorm.DB) tasksvc.TaskRepository {
	return &taskRepository{db}
}

func (t taskRepository) Create(title, description string, userID uint64) (tasksvc.Task, error) {
	task := tasksvc.Task{Title: title, Description: description, Done: false, UserID: userID}
	result := t.db.Create(&task)

	return task, result.Error
}

func (t taskRepository) FindAll(userID uint64) ([]tasksvc.Task, error) {
	var tasks []tasksvc.Task
	result := t.db.Where("user_id = ?", userID).Find(&tasks)

	return tasks, result.Error
}

func (t taskRepository) Find(userID, taskID uint64) (tasksvc.Task, error) {
	var task tasksvc.Task
	result := t.db.Where("id = ? AND user_id = ?", taskID, userID).First(&task)

	return task, result.Error
}

func (t taskRepository) Update(task tasksvc.Task) (tasksvc.Task, error) {
	tk, err := t.Find(task.UserID, task.ID)
	if err != nil {
		return tasksvc.Task{}, err
	}

	result := t.db.Model(&tk).Updates(
		map[string]interface{}{
			"title":       task.Title,
			"description": task.Description,
			"done":        task.Done,
			"user_id":     task.UserID,
		})
	if result.Error != nil {
		return tasksvc.Task{}, err
	}

	return tk, err
}

func (t taskRepository) Delete(userID, taskID uint64) (bool, error) {
	task := tasksvc.Task{ID: taskID}
	result := t.db.Where("user_id", userID).Delete(&task)

	if result.Error != nil {
		return false, result.Error
	}
	return true, nil
}
