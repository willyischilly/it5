package models

import "time"

var ValidContourNames = map[string]bool{
	"Dev": true, "Qa": true, "Uat": true, "Prod": true,
}

type DeploymentContour struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;uniqueIndex;not null" json:"name"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

func (DeploymentContour) TableName() string { return "deployment_contours" }
