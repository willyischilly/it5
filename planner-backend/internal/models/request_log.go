package models

import "time"

const (
	RequestLogActionCreated          = "plan_created"
	RequestLogActionSubmitted        = "plan_submitted"
	RequestLogActionStatusChanged    = "plan_status_changed"
	RequestLogActionDeleted          = "plan_deleted"
	RequestLogActionDeadlineExtended = "deadline_extended"
)

type RequestLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RequestID uint      `gorm:"not null;index" json:"request_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	Action    string    `gorm:"size:50;not null" json:"action"`
	OldStatus *string   `gorm:"size:50" json:"old_status,omitempty"`
	NewStatus *string   `gorm:"size:50" json:"new_status,omitempty"`
	Details   string    `gorm:"type:text" json:"details,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (RequestLog) TableName() string { return "request_logs" }
