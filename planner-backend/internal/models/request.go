package models

import "time"

const (
	RequestStatusDraft      = "draft"
	RequestStatusSubmitted  = "submitted"
	RequestStatusInProgress = "in_progress"
	RequestStatusCompleted  = "completed"
	RequestStatusOverdue    = "overdue"
)

type Request struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Title      string     `gorm:"size:255;not null" json:"title"`
	CustomerID uint       `gorm:"not null" json:"customer_id"`
	ContourID  uint       `gorm:"not null" json:"contour_id"`
	Status     string     `gorm:"size:50;default:draft" json:"status"`
	// DeadlineAt — срок всей заявки; просрочка (status overdue) только у заявки, не у задач.
	DeadlineAt *time.Time `json:"deadline_at,omitempty"`
	TotalHours int        `gorm:"default:0" json:"total_hours"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	Customer *User              `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Contour  *DeploymentContour `gorm:"foreignKey:ContourID" json:"contour,omitempty"`
	Tasks    []Task             `gorm:"foreignKey:RequestID" json:"tasks,omitempty"`
}

func (Request) TableName() string { return "requests" }
