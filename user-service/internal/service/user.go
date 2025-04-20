package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ramisback/istio-rate-limiter/user-service/internal/models"
	"github.com/ramisback/istio-rate-limiter/user-service/internal/repository"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already taken")
)

// UserService defines the interface for user-related operations
type UserService interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id string) error
	ValidateCredentials(ctx context.Context, email, password string) (*models.User, error)
}

type userService struct {
	logger *zap.Logger
	repo   repository.UserRepository
}

// NewUserService creates a new instance of UserService
func NewUserService(logger *zap.Logger, repo repository.UserRepository) UserService {
	return &userService{
		logger: logger,
		repo:   repo,
	}
}

func (s *userService) CreateUser(ctx context.Context, user *models.User) error {
	// Check if email is already taken
	existingUser, err := s.repo.GetByEmail(ctx, user.Email)
	if err == nil && existingUser != nil {
		return ErrEmailTaken
	}

	// Set creation time and ID
	now := time.Now()
	user.ID = uuid.New().String()
	user.CreatedAt = now
	user.UpdatedAt = now

	return s.repo.Create(ctx, user)
}

func (s *userService) GetUser(ctx context.Context, id string) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *userService) UpdateUser(ctx context.Context, user *models.User) error {
	// Check if user exists
	existingUser, err := s.repo.GetByID(ctx, user.ID)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return ErrUserNotFound
	}

	// Update timestamp
	user.UpdatedAt = time.Now()

	return s.repo.Update(ctx, user)
}

func (s *userService) DeleteUser(ctx context.Context, id string) error {
	// Check if user exists
	existingUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return ErrUserNotFound
	}

	return s.repo.Delete(ctx, id)
}

func (s *userService) ValidateCredentials(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if !user.ValidatePassword(password) {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
