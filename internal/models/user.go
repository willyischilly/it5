package models

import "time"

const (
	RoleAdmin    = "admin"
	RoleCustomer = "customer"
	RoleExecutor = "executor"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	Role      string    `gorm:"size:50;not null" json:"role"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }

func ValidRole(role string) bool {
	switch role {
	case RoleAdmin, RoleCustomer, RoleExecutor:
		return true
	default:
		return false
	}
}

func ValidRegisterRole(role string) bool {
	return role == RoleCustomer || role == RoleExecutor
}
