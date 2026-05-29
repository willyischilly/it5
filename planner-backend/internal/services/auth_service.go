package services

import (
	"errors"
	"time"

	"planner-backend/internal/config"
	"planner-backend/internal/models"
	"planner-backend/internal/repositories"
	"planner-backend/pkg/auth"
	"planner-backend/pkg/validation"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	users *repositories.UserRepository
	cfg   *config.Config
}

func NewAuthService(users *repositories.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{users: users, cfg: cfg}
}

type RegisterInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	Patronymic string `json:"patronymic"`
	Role       string `json:"role"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResult struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID         uint   `json:"id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	Patronymic string `json:"patronymic"`
	FullName   string `json:"full_name"`
}

func ToUserResponse(u *models.User) UserResponse {
	return UserResponse{
		ID:         u.ID,
		Email:      u.Email,
		Role:       u.Role,
		LastName:   u.LastName,
		FirstName:  u.FirstName,
		Patronymic: u.Patronymic,
		FullName:   u.FullName(),
	}
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) Register(in RegisterInput) (*LoginResult, error) {
	if !validation.Email(in.Email) {
		return nil, errors.New("invalid email format")
	}
	if !validation.Password(in.Password) {
		return nil, errors.New("password must be at least 6 characters")
	}
	person := trimPerson(PersonInput{
		LastName: in.LastName, FirstName: in.FirstName, Patronymic: in.Patronymic,
	})
	if err := validatePerson(person); err != nil {
		return nil, err
	}
	if !models.ValidRegisterRole(in.Role) {
		return nil, errors.New("role must be customer or executor")
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
		Email:    in.Email,
		Password: hash,
		Role:     in.Role,
	}
	applyPersonToUser(user, person)
	if err := s.users.Create(user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, User: ToUserResponse(user)}, nil
}

func (s *AuthService) Me(userID uint) (*UserResponse, error) {
	user, err := s.users.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	resp := ToUserResponse(user)
	return &resp, nil
}

func (s *AuthService) Login(in LoginInput) (*LoginResult, error) {
	if !validation.Email(in.Email) {
		return nil, errors.New("invalid email or password")
	}

	user, err := s.users.FindByEmail(in.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token: token,
		User:  ToUserResponse(user),
	}, nil
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.cfg.JWTExpireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
