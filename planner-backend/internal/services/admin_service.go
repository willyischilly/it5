package services

import (
	"errors"

	"strings"

	"planner-backend/internal/models"
	"planner-backend/internal/repositories"
	"planner-backend/pkg/auth"
	"planner-backend/pkg/validation"

	"gorm.io/gorm"
)

type AdminService struct {
	users       *repositories.UserRepository
	works       *repositories.WorkRepository
	contours    *repositories.ContourRepository
	tasks       *repositories.TaskRepository
	requestLogs *repositories.RequestLogRepository
}

func NewAdminService(
	users *repositories.UserRepository,
	works *repositories.WorkRepository,
	contours *repositories.ContourRepository,
	tasks *repositories.TaskRepository,
	requestLogs *repositories.RequestLogRepository,
) *AdminService {
	return &AdminService{
		users: users, works: works, contours: contours,
		tasks: tasks, requestLogs: requestLogs,
	}
}

func auditLimit(raw int) int {
	if raw <= 0 {
		return 100
	}
	if raw > 500 {
		return 500
	}
	return raw
}

func (s *AdminService) ListRequestLogs(requestID *uint, limit int) ([]models.RequestLog, error) {
	return s.requestLogs.List(requestID, auditLimit(limit))
}

func (s *AdminService) ListTaskLogs(taskID, requestID *uint, limit int) ([]models.TaskLog, error) {
	return s.tasks.ListLogs(taskID, requestID, auditLimit(limit))
}

func (s *AdminService) ListUsers() ([]UserResponse, error) {
	users, err := s.users.List()
	if err != nil {
		return nil, err
	}
	out := make([]UserResponse, len(users))
	for i := range users {
		out[i] = ToUserResponse(&users[i])
	}
	return out, nil
}

type CreateUserInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

func (s *AdminService) CreateUser(in CreateUserInput) (*UserResponse, error) {
	if !validation.Email(in.Email) {
		return nil, errors.New("invalid email format")
	}
	if !validation.Password(in.Password) {
		return nil, errors.New("password must be at least 6 characters")
	}
	if !validation.NonEmpty(in.Name) {
		return nil, errors.New("name is required")
	}
	if !models.ValidRole(in.Role) {
		return nil, errors.New("invalid role")
	}
	exists, err := s.users.EmailExists(in.Email, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email already exists")
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email: in.Email, Password: hash, Name: in.Name, Role: in.Role,
	}
	if err := s.users.Create(user); err != nil {
		return nil, err
	}
	resp := ToUserResponse(user)
	return &resp, nil
}

type UpdateUserInput struct {
	Email    *string `json:"email"`
	Password *string `json:"password"`
	Name     *string `json:"name"`
	Role     *string `json:"role"`
}

func (s *AdminService) UpdateUser(id uint, in UpdateUserInput) (*UserResponse, error) {
	user, err := s.users.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if in.Email != nil {
		if !validation.Email(*in.Email) {
			return nil, errors.New("invalid email format")
		}
		exists, err := s.users.EmailExists(*in.Email, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.New("email already exists")
		}
		user.Email = *in.Email
	}
	if in.Password != nil {
		if !validation.Password(*in.Password) {
			return nil, errors.New("password must be at least 6 characters")
		}
		hash, err := auth.HashPassword(*in.Password)
		if err != nil {
			return nil, err
		}
		user.Password = hash
	}
	if in.Name != nil {
		if !validation.NonEmpty(*in.Name) {
			return nil, errors.New("name is required")
		}
		user.Name = *in.Name
	}
	if in.Role != nil {
		if !models.ValidRole(*in.Role) {
			return nil, errors.New("invalid role")
		}
		user.Role = *in.Role
	}

	if err := s.users.Update(user); err != nil {
		return nil, err
	}
	resp := ToUserResponse(user)
	return &resp, nil
}

func (s *AdminService) DeleteUser(id uint) error {
	user, err := s.users.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}
	if user.Role == models.RoleAdmin {
		return errors.New("cannot delete admin user")
	}
	if err := s.tasks.ClearExecutorAssignments(id); err != nil {
		return err
	}
	if err := s.tasks.DeleteLogsByUser(id); err != nil {
		return err
	}
	if err := s.requestLogs.DeleteByUser(id); err != nil {
		return err
	}
	return s.users.Delete(id)
}

func (s *AdminService) ListWorks() ([]models.Work, error) {
	return s.works.List()
}

type WorkInput struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	NormativeHours int    `json:"normative_hours"`
}

func (s *AdminService) validateWorkInput(in WorkInput) error {
	if !validation.NonEmpty(in.Name) {
		return errors.New("work name is required")
	}
	if !validation.PositiveInt(in.NormativeHours) {
		return errors.New("normative_hours must be at least 1")
	}
	return nil
}

func (s *AdminService) CreateWork(in WorkInput) (*models.Work, error) {
	if err := s.validateWorkInput(in); err != nil {
		return nil, err
	}
	work := &models.Work{
		Name: in.Name, Description: in.Description, NormativeHours: in.NormativeHours,
	}
	if err := s.works.Create(work); err != nil {
		return nil, err
	}
	return work, nil
}

type UpdateWorkInput struct {
	Name           *string `json:"name"`
	Description    *string `json:"description"`
	NormativeHours *int    `json:"normative_hours"`
}

func (s *AdminService) UpdateWork(id uint, in UpdateWorkInput) (*models.Work, error) {
	work, err := s.works.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("work not found")
		}
		return nil, err
	}

	if in.Name != nil {
		work.Name = *in.Name
	}
	if in.Description != nil {
		work.Description = *in.Description
	}
	if in.NormativeHours != nil {
		work.NormativeHours = *in.NormativeHours
	}

	if err := s.validateWorkInput(WorkInput{
		Name: work.Name, NormativeHours: work.NormativeHours,
	}); err != nil {
		return nil, err
	}

	if err := s.works.Update(work); err != nil {
		return nil, err
	}
	return work, nil
}

func (s *AdminService) DeleteWork(id uint) error {
	_, err := s.works.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("work not found")
		}
		return err
	}
	inUse, err := s.works.InUse(id)
	if err != nil {
		return err
	}
	if inUse {
		return errors.New("work is used in requests and cannot be deleted")
	}
	return s.works.Delete(id)
}

func (s *AdminService) ListContours() ([]models.DeploymentContour, error) {
	return s.contours.List()
}

type ContourInput struct {
	Name string `json:"name"`
}

func (s *AdminService) validateContourName(name string) error {
	if !validation.MaxLen(name, 50) {
		return errors.New("contour name is required and must be at most 50 characters")
	}
	return nil
}

func (s *AdminService) CreateContour(in ContourInput) (*models.DeploymentContour, error) {
	if err := s.validateContourName(in.Name); err != nil {
		return nil, err
	}
	exists, err := s.contours.NameExists(in.Name, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("contour name already exists")
	}
	c := &models.DeploymentContour{Name: strings.TrimSpace(in.Name)}
	if err := s.contours.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *AdminService) UpdateContour(id uint, in ContourInput) (*models.DeploymentContour, error) {
	c, err := s.contours.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("contour not found")
		}
		return nil, err
	}
	if err := s.validateContourName(in.Name); err != nil {
		return nil, err
	}
	exists, err := s.contours.NameExists(in.Name, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("contour name already exists")
	}
	c.Name = strings.TrimSpace(in.Name)
	if err := s.contours.Update(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *AdminService) DeleteContour(id uint) error {
	_, err := s.contours.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("contour not found")
		}
		return err
	}
	inUse, err := s.contours.InUse(id)
	if err != nil {
		return err
	}
	if inUse {
		return errors.New("contour is used in requests and cannot be deleted")
	}
	return s.contours.Delete(id)
}
