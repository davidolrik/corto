package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/davidolrik/corto/internal/auth"
	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/google/uuid"
	"github.com/zitadel/passwap"
	"github.com/zitadel/passwap/bcrypt"
)

type UserService struct {
	log       *slog.Logger
	db        core.Database
	passwords *passwap.Swapper
}

func NewUserService(log *slog.Logger, db core.Database) *UserService {
	return &UserService{
		log: log,
		db:  db,
		passwords: passwap.NewSwapper(
			bcrypt.New(bcrypt.DefaultCost, nil),
		),
	}
}

// CreateUser creates a user with a hashed password. Usernames are unique.
func (s *UserService) CreateUser(ctx context.Context, username, password string) (*model.User, error) {
	if username == "" {
		return nil, fmt.Errorf("username must not be empty")
	}
	if password == "" {
		return nil, fmt.Errorf("password must not be empty")
	}

	exists, err := s.db.NewSelect().Model((*model.User)(nil)).
		Where("username = ?", username).
		Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking username %q: %w", username, err)
	}
	if exists {
		return nil, fmt.Errorf("user %q already exists", username)
	}

	hash, err := s.passwords.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now().Truncate(time.Second)
	user := &model.User{
		PublicID:  uuid.New().String(),
		Username:  username,
		Password:  hash,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = s.db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}
	return user, nil
}

// authenticatedUser loads the user identified by the auth claims in the context.
func (s *UserService) authenticatedUser(ctx context.Context) (*model.User, error) {
	claims := auth.GetClaims(ctx)
	if claims.UserPublicID == "" {
		return nil, fmt.Errorf("not authenticated")
	}
	user := &model.User{}
	err := s.db.NewSelect().Model(user).Where("public_id = ?", claims.UserPublicID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("user %q not found: %w", claims.UserPublicID, err)
	}
	return user, nil
}

// GetProfile returns the authenticated user's profile.
func (s *UserService) GetProfile(ctx context.Context) (*handlers.ProfileData, error) {
	user, err := s.authenticatedUser(ctx)
	if err != nil {
		return nil, err
	}
	return &handlers.ProfileData{
		PublicID: user.PublicID,
		Username: user.Username,
	}, nil
}

// ChangePassword verifies the current password and replaces it with the new one.
func (s *UserService) ChangePassword(ctx context.Context, currentPassword, newPassword string) error {
	user, err := s.authenticatedUser(ctx)
	if err != nil {
		return err
	}

	if _, err := s.passwords.Verify(user.Password, currentPassword); err != nil {
		return fmt.Errorf("verifying current password: %w", handlers.ErrInvalidCredentials)
	}

	hash, err := s.passwords.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user.Password = hash
	user.UpdatedAt = time.Now().Truncate(time.Second)
	_, err = s.db.NewUpdate().Model(user).Column("password", "updated_at").WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	return nil
}
