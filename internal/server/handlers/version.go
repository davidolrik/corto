package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// VersionOutput is the response for the version endpoint.
type VersionOutput struct {
	Body struct {
		Version string `json:"version" doc:"Corto version"`
	}
}

// RegisterVersionRoutes registers the public version endpoint on the given
// Huma API. An empty version (untagged development build) reports "devel".
func RegisterVersionRoutes(api huma.API, version string) {
	if version == "" {
		version = "devel"
	}
	huma.Register(api, huma.Operation{
		OperationID: "get-version",
		Method:      http.MethodGet,
		Path:        "/api/version",
		Summary:     "Get the Corto version",
		Tags:        []string{"Meta"},
	}, func(ctx context.Context, input *struct{}) (*VersionOutput, error) {
		out := &VersionOutput{}
		out.Body.Version = version
		return out, nil
	})
}
