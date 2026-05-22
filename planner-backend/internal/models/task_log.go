package models

import "time"

type TaskLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    uint      `gorm:"not null" json:"task_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	OldStatus *string   `gorm:"size:50" json:"old_status"`
	NewStatus string    `gorm:"size:50;not null" json:"new_status"`
	CreatedAt time.Time `json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Task *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
}

func (TaskLog) TableName() string { return "task_logs" }
