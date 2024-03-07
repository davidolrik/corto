package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// ProfileStore defines the interface for the authenticated user's profile.
type ProfileStore interface {
	GetProfile(ctx context.Context) (*ProfileData, error)
	ChangePassword(ctx context.Context, currentPassword, newPassword string) error
}

// ProfileData represents the authenticated user in the service layer.
type ProfileData struct {
	PublicID string
	Username string
}

// ProfileOutput is the response for the profile endpoint.
type ProfileOutput struct {
	Body struct {
		UserID   string `json:"user_id" doc:"Public ID of the user"`
		Username string `json:"username" doc:"Username"`
	}
}

// ChangePasswordInput is the request body for changing the password.
type ChangePasswordInput struct {
	Body struct {
		CurrentPassword string `json:"current_password" required:"true" doc:"Current password"`
		NewPassword     string `json:"new_password" required:"true" minLength:"8" doc:"New password"`
	}
}

// RegisterProfileRoutes registers the profile endpoints on the given Huma API.
func RegisterProfileRoutes(api huma.API, store ProfileStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-profile",
		Method:      http.MethodGet,
		Path:        "/api/profile",
		Summary:     "Get the authenticated user's profile",
		Tags:        []string{"Profile"},
	}, func(ctx context.Context, input *struct{}) (*ProfileOutput, error) {
		profile, err := store.GetProfile(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load profile", err)
		}
		out := &ProfileOutput{}
		out.Body.UserID = profile.PublicID
		out.Body.Username = profile.Username
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "change-password",
		Method:        http.MethodPut,
		Path:          "/api/profile/password",
		Summary:       "Change the authenticated user's password",
		Tags:          []string{"Profile"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *ChangePasswordInput) (*struct{}, error) {
		err := store.ChangePassword(ctx, input.Body.CurrentPassword, input.Body.NewPassword)
		if errors.Is(err, ErrInvalidCredentials) {
			return nil, huma.Error403Forbidden("current password is incorrect")
		}
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to change password", err)
		}
		return nil, nil
	})
}
