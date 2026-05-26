package models

import "time"

const (
	TaskStatusPending    = "pending"
	TaskStatusInProgress = "in_progress"
	TaskStatusCompleted  = "completed"
)

type Task struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	RequestID       uint       `gorm:"not null" json:"request_id"`
	WorkID          uint       `gorm:"not null" json:"work_id"`
	ExecutorID      *uint      `json:"executor_id"`
	Status          string     `gorm:"size:50;default:pending" json:"status"`
	CustomerComment string     `gorm:"type:text" json:"customer_comment,omitempty"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	Work     *Work    `gorm:"foreignKey:WorkID" json:"work,omitempty"`
	Executor *User    `gorm:"foreignKey:ExecutorID" json:"executor,omitempty"`
	Request  *Request `gorm:"foreignKey:RequestID" json:"request,omitempty"`
}

func (Task) TableName() string { return "tasks" }

func ValidTaskTransition(oldStatus, newStatus string) bool {
	if oldStatus == newStatus {
		return false
	}
	switch oldStatus {
	case TaskStatusPending:
		return newStatus == TaskStatusInProgress
	case TaskStatusInProgress:
		return newStatus == TaskStatusCompleted
	default:
		return false
	}
}
