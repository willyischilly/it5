package services

import (
	"errors"
	"strings"

	"planner-backend/internal/models"
	"planner-backend/pkg/validation"
)

type PersonInput struct {
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	Patronymic string `json:"patronymic"`
}

func trimPerson(in PersonInput) PersonInput {
	return PersonInput{
		LastName:   strings.TrimSpace(in.LastName),
		FirstName:  strings.TrimSpace(in.FirstName),
		Patronymic: strings.TrimSpace(in.Patronymic),
	}
}

func validatePerson(in PersonInput) error {
	if !validation.PersonName(in.LastName) {
		return errors.New("last_name (фамилия) is required")
	}
	if !validation.PersonName(in.FirstName) {
		return errors.New("first_name (имя) is required")
	}
	if !validation.PersonName(in.Patronymic) {
		return errors.New("patronymic (отчество) is required")
	}
	return nil
}

func applyPersonToUser(user *models.User, in PersonInput) {
	user.LastName = in.LastName
	user.FirstName = in.FirstName
	user.Patronymic = in.Patronymic
}
