package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"ntf-auction-backend/internal/apperr"
	"ntf-auction-backend/internal/model"
	"ntf-auction-backend/internal/repository"
)

type AuthService struct {
	users       repository.UserRepository
	jwtSecret   []byte
	expireHours int
}

type LoginInput struct {
	Username string
	Password string
}

type LoginOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewAuthService(users repository.UserRepository, jwtSecret string, expireHours int) *AuthService {
	if expireHours <= 0 {
		expireHours = 24
	}
	return &AuthService{users: users, jwtSecret: []byte(jwtSecret), expireHours: expireHours}
}

func HashPassword(raw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (s *AuthService) EnsureUser(ctx context.Context, username, rawPassword string) error {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(rawPassword) == "" {
		return apperr.InvalidArgument("default user is not configured")
	}

	_, err := s.users.GetByUsername(ctx, username)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return apperr.Internal("query default user failed")
	}

	hash, hashErr := HashPassword(rawPassword)
	if hashErr != nil {
		return apperr.Internal("hash password failed")
	}

	if createErr := s.users.Create(ctx, model.User{Username: username, PasswordHash: hash}); createErr != nil {
		return apperr.Internal("create default user failed")
	}
	return nil
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (LoginOutput, error) {
	if strings.TrimSpace(in.Username) == "" || strings.TrimSpace(in.Password) == "" {
		return LoginOutput{}, apperr.InvalidArgument("username and password are required")
	}

	user, err := s.users.GetByUsername(ctx, in.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return LoginOutput{}, apperr.Unauthorized("invalid username or password")
		}
		return LoginOutput{}, apperr.Internal("query user failed")
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); compareErr != nil {
		return LoginOutput{}, apperr.Unauthorized("invalid username or password")
	}

	expiresAt := time.Now().Add(time.Duration(s.expireHours) * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      expiresAt.Unix(),
	})

	signed, signErr := token.SignedString(s.jwtSecret)
	if signErr != nil {
		return LoginOutput{}, apperr.Internal("sign jwt token failed")
	}

	return LoginOutput{Token: signed, ExpiresAt: expiresAt}, nil
}
