package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// TagStore defines the interface for tag persistence operations.
type TagStore interface {
	CreateTag(ctx context.Context, t *TagData) (*TagData, error)
	GetTag(ctx context.Context, publicID string) (*TagData, error)
	ListTags(ctx context.Context) ([]*TagData, error)
	UpdateTag(ctx context.Context, publicID string, t *TagData) (*TagData, error)
	PatchTag(ctx context.Context, publicID string, patch *TagPatch) (*TagData, error)
	DeleteTag(ctx context.Context, publicID string) error
}

// TagData represents a tag in the service layer.
type TagData struct {
	PublicID    string
	Slug        string
	Color       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TagPatch represents a partial update to a tag.
type TagPatch struct {
	Slug        *string
	Color       *string
	Description *string
}

// CreateTagInput is the request body for creating a tag.
type CreateTagInput struct {
	Body struct {
		Slug        string `json:"slug" required:"true" doc:"Tag slug"`
		Color       string `json:"color,omitempty" pattern:"^#[0-9a-fA-F]{6}$" doc:"Tag color as #rrggbb"`
		Description string `json:"description,omitempty" doc:"Tag description"`
	}
}

// TagOutput is the response body for a single tag.
type TagOutput struct {
	Body TagBody
}

// TagBody is the JSON body of a tag response.
type TagBody struct {
	PublicID    string    `json:"public_id" doc:"Public identifier"`
	Slug        string    `json:"slug" doc:"Tag slug"`
	Color       string    `json:"color,omitempty" doc:"Tag color as #rrggbb"`
	Description string    `json:"description,omitempty" doc:"Tag description"`
	CreatedAt   time.Time `json:"created_at" doc:"Creation timestamp"`
	UpdatedAt   time.Time `json:"updated_at" doc:"Last update timestamp"`
}

// GetTagInput is the request for fetching a single tag.
type GetTagInput struct {
	PublicID string `path:"id" doc:"Public ID of the tag"`
}

// UpdateTagInput is the request for fully replacing a tag.
type UpdateTagInput struct {
	PublicID string `path:"id" doc:"Public ID of the tag"`
	Body     struct {
		Slug        string `json:"slug" required:"true" doc:"Tag slug"`
		Color       string `json:"color,omitempty" pattern:"^#[0-9a-fA-F]{6}$" doc:"Tag color as #rrggbb"`
		Description string `json:"description,omitempty" doc:"Tag description"`
	}
}

// PatchTagInput is the request for partially updating a tag.
type PatchTagInput struct {
	PublicID string `path:"id" doc:"Public ID of the tag"`
	Body     struct {
		Slug        *string `json:"slug,omitempty" doc:"Tag slug"`
		Color       *string `json:"color,omitempty" pattern:"^#[0-9a-fA-F]{6}$" doc:"Tag color as #rrggbb"`
		Description *string `json:"description,omitempty" doc:"Tag description"`
	}
}

// DeleteTagInput is the request for deleting a tag.
type DeleteTagInput struct {
	PublicID string `path:"id" doc:"Public ID of the tag"`
}

// ListTagsOutput is the response for listing tags.
type ListTagsOutput struct {
	Body []TagBody
}

func tagDataToBody(d *TagData) TagBody {
	return TagBody{
		PublicID:    d.PublicID,
		Slug:        d.Slug,
		Color:       d.Color,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// RegisterTagRoutes registers all tag CRUD operations on the given Huma API.
func RegisterTagRoutes(api huma.API, store TagStore) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-tag",
		Method:        http.MethodPost,
		Path:          "/api/tags",
		Summary:       "Create a tag",
		Tags:          []string{"Tags"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *CreateTagInput) (*TagOutput, error) {
		data := &TagData{
			Slug:        input.Body.Slug,
			Color:       input.Body.Color,
			Description: input.Body.Description,
		}
		created, err := store.CreateTag(ctx, data)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create tag", err)
		}
		return &TagOutput{Body: tagDataToBody(created)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-tag",
		Method:      http.MethodGet,
		Path:        "/api/tags/{id}",
		Summary:     "Get a tag",
		Tags:        []string{"Tags"},
	}, func(ctx context.Context, input *GetTagInput) (*TagOutput, error) {
		data, err := store.GetTag(ctx, input.PublicID)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("tag %q not found", input.PublicID))
		}
		return &TagOutput{Body: tagDataToBody(data)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-tags",
		Method:      http.MethodGet,
		Path:        "/api/tags",
		Summary:     "List all tags",
		Tags:        []string{"Tags"},
	}, func(ctx context.Context, input *struct{}) (*ListTagsOutput, error) {
		tags, err := store.ListTags(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list tags", err)
		}
		bodies := make([]TagBody, len(tags))
		for i, t := range tags {
			bodies[i] = tagDataToBody(t)
		}
		return &ListTagsOutput{Body: bodies}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-tag",
		Method:      http.MethodPut,
		Path:        "/api/tags/{id}",
		Summary:     "Update a tag",
		Tags:        []string{"Tags"},
	}, func(ctx context.Context, input *UpdateTagInput) (*TagOutput, error) {
		data := &TagData{
			Slug:        input.Body.Slug,
			Color:       input.Body.Color,
			Description: input.Body.Description,
		}
		updated, err := store.UpdateTag(ctx, input.PublicID, data)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("tag %q not found", input.PublicID))
		}
		return &TagOutput{Body: tagDataToBody(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "patch-tag",
		Method:      http.MethodPatch,
		Path:        "/api/tags/{id}",
		Summary:     "Partially update a tag",
		Tags:        []string{"Tags"},
	}, func(ctx context.Context, input *PatchTagInput) (*TagOutput, error) {
		patch := &TagPatch{
			Slug:        input.Body.Slug,
			Color:       input.Body.Color,
			Description: input.Body.Description,
		}
		updated, err := store.PatchTag(ctx, input.PublicID, patch)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("tag %q not found", input.PublicID))
		}
		return &TagOutput{Body: tagDataToBody(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-tag",
		Method:        http.MethodDelete,
		Path:          "/api/tags/{id}",
		Summary:       "Delete a tag",
		Tags:          []string{"Tags"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *DeleteTagInput) (*struct{}, error) {
		err := store.DeleteTag(ctx, input.PublicID)
		if err != nil {
			return nil, huma.Error404NotFound(fmt.Sprintf("tag %q not found", input.PublicID))
		}
		return nil, nil
	})
}
