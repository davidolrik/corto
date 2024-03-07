package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// DomainStore defines the interface for domain persistence operations.
type DomainStore interface {
	CreateDomain(ctx context.Context, d *DomainData) (*DomainData, error)
	GetDomain(ctx context.Context, publicID string) (*DomainData, error)
	ListDomains(ctx context.Context) ([]*DomainData, error)
	UpdateDomain(ctx context.Context, publicID string, d *DomainData) (*DomainData, error)
	PatchDomain(ctx context.Context, publicID string, patch *DomainPatch) (*DomainData, error)
	DeleteDomain(ctx context.Context, publicID string) error
}

// DomainData represents a domain in the service layer.
type DomainData struct {
	PublicID    string
	FQDN        string
	FallbackURL string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DomainPatch represents a partial update to a domain.
type DomainPatch struct {
	FQDN        *string
	FallbackURL *string
	Description *string
}

// CreateDomainInput is the request body for creating a domain.
type CreateDomainInput struct {
	Body struct {
		FQDN        string `json:"fqdn" required:"true" doc:"Fully qualified domain name"`
		FallbackURL string `json:"fallback_url,omitempty" doc:"Default fallback URL for this domain"`
		Description string `json:"description,omitempty" doc:"Domain description"`
	}
}

// DomainOutput is the response body for a single domain.
type DomainOutput struct {
	Body DomainBody
}

// DomainBody is the JSON body of a domain response.
type DomainBody struct {
	PublicID    string    `json:"public_id" doc:"Public identifier"`
	FQDN        string    `json:"fqdn" doc:"Fully qualified domain name"`
	FallbackURL string    `json:"fallback_url,omitempty" doc:"Default fallback URL"`
	Description string    `json:"description,omitempty" doc:"Domain description"`
	CreatedAt   time.Time `json:"created_at" doc:"Creation timestamp"`
	UpdatedAt   time.Time `json:"updated_at" doc:"Last update timestamp"`
}

// GetDomainInput is the request for fetching a single domain.
type GetDomainInput struct {
	PublicID string `path:"id" doc:"Public ID of the domain"`
}

// UpdateDomainInput is the request for fully replacing a domain.
type UpdateDomainInput struct {
	PublicID string `path:"id" doc:"Public ID of the domain"`
	Body     struct {
		FQDN        string `json:"fqdn" required:"true" doc:"Fully qualified domain name"`
		FallbackURL string `json:"fallback_url,omitempty" doc:"Default fallback URL for this domain"`
		Description string `json:"description,omitempty" doc:"Domain description"`
	}
}

// PatchDomainInput is the request for partially updating a domain.
type PatchDomainInput struct {
	PublicID string `path:"id" doc:"Public ID of the domain"`
	Body     struct {
		FQDN        *string `json:"fqdn,omitempty" doc:"Fully qualified domain name"`
		FallbackURL *string `json:"fallback_url,omitempty" doc:"Default fallback URL"`
		Description *string `json:"description,omitempty" doc:"Domain description"`
	}
}

// DeleteDomainInput is the request for deleting a domain.
type DeleteDomainInput struct {
	PublicID string `path:"id" doc:"Public ID of the domain"`
}

// ListDomainsOutput is the response for listing domains.
type ListDomainsOutput struct {
	Body []DomainBody
}

func domainDataToBody(d *DomainData) DomainBody {
	return DomainBody{
		PublicID:    d.PublicID,
		FQDN:        d.FQDN,
		FallbackURL: d.FallbackURL,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// RegisterDomainRoutes registers all domain CRUD operations on the given Huma API.
func RegisterDomainRoutes(api huma.API, store DomainStore) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-domain",
		Method:        http.MethodPost,
		Path:          "/api/domains",
		Summary:       "Create a domain",
		Tags:          []string{"Domains"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *CreateDomainInput) (*DomainOutput, error) {
		data := &DomainData{
			FQDN:        input.Body.FQDN,
			FallbackURL: input.Body.FallbackURL,
			Description: input.Body.Description,
		}
		created, err := store.CreateDomain(ctx, data)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create domain", err)
		}
		return &DomainOutput{Body: domainDataToBody(created)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-domain",
		Method:      http.MethodGet,
		Path:        "/api/domains/{id}",
		Summary:     "Get a domain",
		Tags:        []string{"Domains"},
	}, func(ctx context.Context, input *GetDomainInput) (*DomainOutput, error) {
		data, err := store.GetDomain(ctx, input.PublicID)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("domain %q not found", input.PublicID))
		}
		return &DomainOutput{Body: domainDataToBody(data)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-domains",
		Method:      http.MethodGet,
		Path:        "/api/domains",
		Summary:     "List all domains",
		Tags:        []string{"Domains"},
	}, func(ctx context.Context, input *struct{}) (*ListDomainsOutput, error) {
		domains, err := store.ListDomains(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list domains", err)
		}
		bodies := make([]DomainBody, len(domains))
		for i, d := range domains {
			bodies[i] = domainDataToBody(d)
		}
		return &ListDomainsOutput{Body: bodies}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-domain",
		Method:      http.MethodPut,
		Path:        "/api/domains/{id}",
		Summary:     "Update a domain",
		Tags:        []string{"Domains"},
	}, func(ctx context.Context, input *UpdateDomainInput) (*DomainOutput, error) {
		data := &DomainData{
			FQDN:        input.Body.FQDN,
			FallbackURL: input.Body.FallbackURL,
			Description: input.Body.Description,
		}
		updated, err := store.UpdateDomain(ctx, input.PublicID, data)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("domain %q not found", input.PublicID))
		}
		return &DomainOutput{Body: domainDataToBody(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "patch-domain",
		Method:      http.MethodPatch,
		Path:        "/api/domains/{id}",
		Summary:     "Partially update a domain",
		Tags:        []string{"Domains"},
	}, func(ctx context.Context, input *PatchDomainInput) (*DomainOutput, error) {
		patch := &DomainPatch{
			FQDN:        input.Body.FQDN,
			FallbackURL: input.Body.FallbackURL,
			Description: input.Body.Description,
		}
		updated, err := store.PatchDomain(ctx, input.PublicID, patch)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("domain %q not found", input.PublicID))
		}
		return &DomainOutput{Body: domainDataToBody(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-domain",
		Method:        http.MethodDelete,
		Path:          "/api/domains/{id}",
		Summary:       "Delete a domain",
		Tags:          []string{"Domains"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *DeleteDomainInput) (*struct{}, error) {
		err := store.DeleteDomain(ctx, input.PublicID)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("domain %q not found", input.PublicID))
		}
		return nil, nil
	})
}
