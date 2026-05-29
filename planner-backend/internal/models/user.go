package models

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	RoleAdmin    = "admin"
	RoleCustomer = "customer"
	RoleExecutor = "executor"
)

type User struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Email      string    `gorm:"uniqueIndex;size:255;not null" json:"email"`
	Password   string    `gorm:"size:255;not null" json:"-"`
	Role       string    `gorm:"size:50;not null" json:"role"`
	LastName   string    `gorm:"size:100;not null" json:"last_name"`
	FirstName  string    `gorm:"size:100;not null" json:"first_name"`
	Patronymic string    `gorm:"size:100;not null" json:"patronymic"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }

func (u *User) FullName() string {
	return strings.TrimSpace(u.LastName + " " + u.FirstName + " " + u.Patronymic)
}

func (u User) MarshalJSON() ([]byte, error) {
	type userAlias User
	return json.Marshal(struct {
		userAlias
		FullName string `json:"full_name"`
	}{
		userAlias: userAlias(u),
		FullName:  u.FullName(),
	})
}

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
