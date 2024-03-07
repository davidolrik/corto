package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// AuthStore defines the interface for authentication operations.
type AuthStore interface {
	Login(ctx context.Context, username, password, tenantSlug string) (*LoginResult, error)
	SwitchTenant(ctx context.Context, tenantSlug string) (*LoginResult, error)
}

// TenantMembership describes one tenant a user has access to.
type TenantMembership struct {
	Slug    string
	Name    string
	IsAdmin bool
}

// LoginResult holds the result of a successful login or tenant switch.
type LoginResult struct {
	Token          string
	UserPublicID   string
	Username       string
	TenantPublicID string
	TenantSlug     string
	TenantName     string
	IsAdmin        bool
	Tenants        []TenantMembership
}

// LoginInput is the request body for logging in.
type LoginInput struct {
	Body struct {
		Username string `json:"username" required:"true" doc:"Username"`
		Password string `json:"password" required:"true" doc:"Password"`
		Tenant   string `json:"tenant,omitempty" doc:"Slug of the tenant to log in to; defaults to the user's first tenant"`
	}
}

// SwitchTenantInput is the request body for switching the active tenant.
type SwitchTenantInput struct {
	Body struct {
		Tenant string `json:"tenant" required:"true" doc:"Slug of the tenant to switch to"`
	}
}

// LoginOutput is the response body for a successful login.
type LoginOutput struct {
	Body LoginBody
}

// TenantMembershipBody is one tenant in the login response.
type TenantMembershipBody struct {
	Slug    string `json:"slug" doc:"Tenant slug"`
	Name    string `json:"name" doc:"Tenant name"`
	IsAdmin bool   `json:"is_admin" doc:"Whether the user is an admin of this tenant"`
}

// LoginBody is the JSON body of a login response.
type LoginBody struct {
	Token      string                 `json:"token" doc:"PASETO authentication token"`
	UserID     string                 `json:"user_id" doc:"Public ID of the authenticated user"`
	Username   string                 `json:"username" doc:"Username of the authenticated user"`
	TenantID   string                 `json:"tenant_id" doc:"Public ID of the active tenant"`
	TenantSlug string                 `json:"tenant_slug" doc:"Slug of the active tenant"`
	TenantName string                 `json:"tenant_name" doc:"Name of the active tenant"`
	IsAdmin    bool                   `json:"is_admin" doc:"Whether the user is an admin of the active tenant"`
	Tenants    []TenantMembershipBody `json:"tenants" doc:"All tenants the user has access to"`
}

func loginResultToBody(result *LoginResult) LoginBody {
	tenants := make([]TenantMembershipBody, len(result.Tenants))
	for i, t := range result.Tenants {
		tenants[i] = TenantMembershipBody{Slug: t.Slug, Name: t.Name, IsAdmin: t.IsAdmin}
	}
	return LoginBody{
		Token:      result.Token,
		UserID:     result.UserPublicID,
		Username:   result.Username,
		TenantID:   result.TenantPublicID,
		TenantSlug: result.TenantSlug,
		TenantName: result.TenantName,
		IsAdmin:    result.IsAdmin,
		Tenants:    tenants,
	}
}

// RegisterAuthRoutes registers authentication endpoints on the given Huma API.
func RegisterAuthRoutes(api huma.API, store AuthStore) {
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/api/auth/login",
		Summary:     "Log in",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *LoginInput) (*LoginOutput, error) {
		result, err := store.Login(ctx, input.Body.Username, input.Body.Password, input.Body.Tenant)
		if err != nil {
			return nil, huma.Error401Unauthorized("invalid credentials")
		}
		return &LoginOutput{Body: loginResultToBody(result)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "switch-tenant",
		Method:      http.MethodPost,
		Path:        "/api/auth/tenant",
		Summary:     "Switch the active tenant",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *SwitchTenantInput) (*LoginOutput, error) {
		result, err := store.SwitchTenant(ctx, input.Body.Tenant)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("tenant %q not found", input.Body.Tenant))
		}
		return &LoginOutput{Body: loginResultToBody(result)}, nil
	})
}
