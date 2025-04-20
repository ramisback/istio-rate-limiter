package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ramisback/istio-rate-limiter/user-service/internal/models"
	"github.com/redis/go-redis/v9"
)

var (
	ErrDatabaseError = errors.New("database error")
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
}

type userRepository struct {
	client *redis.Client
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(client *redis.Client) UserRepository {
	return &userRepository{
		client: client,
	}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	// Store user by ID
	userKey := fmt.Sprintf("user:%s", user.ID)
	userData, err := json.Marshal(user)
	if err != nil {
		return ErrDatabaseError
	}
	if err := r.client.Set(ctx, userKey, userData, 0).Err(); err != nil {
		return ErrDatabaseError
	}

	// Store user ID by email for lookup
	emailKey := fmt.Sprintf("user:email:%s", user.Email)
	if err := r.client.Set(ctx, emailKey, user.ID, 0).Err(); err != nil {
		return ErrDatabaseError
	}

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	userKey := fmt.Sprintf("user:%s", id)
	userData, err := r.client.Get(ctx, userKey).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, ErrDatabaseError
	}

	var user models.User
	if err := json.Unmarshal(userData, &user); err != nil {
		return nil, ErrDatabaseError
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	emailKey := fmt.Sprintf("user:email:%s", email)
	userID, err := r.client.Get(ctx, emailKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, ErrDatabaseError
	}

	return r.GetByID(ctx, userID)
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	// Check if user exists
	existingUser, err := r.GetByID(ctx, user.ID)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return nil
	}

	// Update user data
	userKey := fmt.Sprintf("user:%s", user.ID)
	userData, err := json.Marshal(user)
	if err != nil {
		return ErrDatabaseError
	}
	if err := r.client.Set(ctx, userKey, userData, 0).Err(); err != nil {
		return ErrDatabaseError
	}

	// Update email index if email changed
	if existingUser.Email != user.Email {
		oldEmailKey := fmt.Sprintf("user:email:%s", existingUser.Email)
		newEmailKey := fmt.Sprintf("user:email:%s", user.Email)

		pipe := r.client.Pipeline()
		pipe.Del(ctx, oldEmailKey)
		pipe.Set(ctx, newEmailKey, user.ID, 0)

		if _, err := pipe.Exec(ctx); err != nil {
			return ErrDatabaseError
		}
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	// Get user to find email
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	// Delete user data and email index
	userKey := fmt.Sprintf("user:%s", id)
	emailKey := fmt.Sprintf("user:email:%s", user.Email)

	pipe := r.client.Pipeline()
	pipe.Del(ctx, userKey)
	pipe.Del(ctx, emailKey)

	if _, err := pipe.Exec(ctx); err != nil {
		return ErrDatabaseError
	}

	return nil
}
