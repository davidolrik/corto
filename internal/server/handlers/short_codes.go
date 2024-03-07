package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// ShortCodeStore defines the interface for short code persistence operations.
type ShortCodeStore interface {
	CreateShortCode(ctx context.Context, sc *ShortCodeData) (*ShortCodeData, error)
	GetShortCode(ctx context.Context, publicID string) (*ShortCodeData, error)
	ListShortCodes(ctx context.Context) ([]*ShortCodeData, error)
	UpdateShortCode(ctx context.Context, publicID string, sc *ShortCodeData) (*ShortCodeData, error)
	PatchShortCode(ctx context.Context, publicID string, patch *ShortCodePatch) (*ShortCodeData, error)
	DeleteShortCode(ctx context.Context, publicID string) error
}

// ShortCodeData represents a short code in the service layer.
type ShortCodeData struct {
	PublicID         string
	Title            string
	Description      string
	Slug             string
	TargetURL        string
	FallbackURL      string
	IsCrawlable      bool
	ForwardQuery     bool
	ValidSince       *time.Time
	ValidUntil       *time.Time
	Domains          []string
	Tags             []string
	Visits           int
	VisitsThisWeek   int
	VisitsByDomain   map[string]int
	VisitsByCampaign map[string]int
	VisitsByCountry  map[string]int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ShortCodePatch represents a partial update to a short code.
// Nil pointer fields are not updated; non-nil fields are applied.
type ShortCodePatch struct {
	Title        *string
	Description  *string
	Slug         *string
	TargetURL    *string
	FallbackURL  *string
	IsCrawlable  *bool
	ForwardQuery *bool
	ValidSince   *time.Time
	ValidUntil   *time.Time
	Domains      *[]string
	Tags         *[]string
}

// CreateShortCodeInput is the request body for creating a short code.
type CreateShortCodeInput struct {
	Body struct {
		Title        string     `json:"title,omitempty" doc:"Title of the short code"`
		Description  string     `json:"description,omitempty" doc:"Short code description"`
		Slug         string     `json:"slug,omitempty" doc:"Short URL slug (auto-generated when omitted)"`
		TargetURL    string     `json:"target_url" required:"true" doc:"Target URL to redirect to"`
		FallbackURL  string     `json:"fallback_url,omitempty" doc:"Fallback URL when link is not valid"`
		IsCrawlable  bool       `json:"is_crawlable,omitempty" doc:"Include in robots.txt"`
		ForwardQuery bool       `json:"forward_query,omitempty" doc:"Forward query string to target"`
		ValidSince   *time.Time `json:"valid_since,omitempty" doc:"Start of validity window"`
		ValidUntil   *time.Time `json:"valid_until,omitempty" doc:"End of validity window"`
		Domains      []string   `json:"domains" required:"true" minItems:"1" doc:"Domain FQDNs where this short code is available"`
		Tags         []string   `json:"tags,omitempty" doc:"Tag slugs to associate with this short code"`
	}
}

// ShortCodeOutput is the response body for a single short code.
type ShortCodeOutput struct {
	Body ShortCodeBody
}

// ShortCodeBody is the JSON body of a short code response.
type ShortCodeBody struct {
	PublicID         string         `json:"public_id" doc:"Public identifier"`
	Title            string         `json:"title,omitempty" doc:"Title of the short code"`
	Description      string         `json:"description,omitempty" doc:"Short code description"`
	Slug             string         `json:"slug" doc:"Short URL slug"`
	TargetURL        string         `json:"target_url" doc:"Target URL"`
	FallbackURL      string         `json:"fallback_url,omitempty" doc:"Fallback URL"`
	IsCrawlable      bool           `json:"is_crawlable" doc:"Include in robots.txt"`
	ForwardQuery     bool           `json:"forward_query" doc:"Forward query string to target"`
	ValidSince       *time.Time     `json:"valid_since,omitempty" doc:"Start of validity window"`
	ValidUntil       *time.Time     `json:"valid_until,omitempty" doc:"End of validity window"`
	Domains          []string       `json:"domains" doc:"Domain FQDNs where this short code is available"`
	Tags             []string       `json:"tags" doc:"Tag slugs associated with this short code"`
	Visits           int            `json:"visits" doc:"Total number of recorded visits"`
	VisitsThisWeek   int            `json:"visits_this_week" doc:"Visits recorded in the last 7 days"`
	VisitsByDomain   map[string]int `json:"visits_by_domain" doc:"Recorded visits per domain FQDN"`
	VisitsByCampaign map[string]int `json:"visits_by_campaign" doc:"Recorded visits per campaign; visits without a campaign count as \"direct\""`
	VisitsByCountry  map[string]int `json:"visits_by_country" doc:"Recorded visits per ISO country code; unresolved countries count as \"unknown\""`
	CreatedAt        time.Time      `json:"created_at" doc:"Creation timestamp"`
	UpdatedAt        time.Time      `json:"updated_at" doc:"Last update timestamp"`
}

// GetShortCodeInput is the request for fetching a single short code.
type GetShortCodeInput struct {
	PublicID string `path:"id" doc:"Public ID of the short code"`
}

// UpdateShortCodeInput is the request for fully replacing a short code.
type UpdateShortCodeInput struct {
	PublicID string `path:"id" doc:"Public ID of the short code"`
	Body     struct {
		Title        string     `json:"title,omitempty" doc:"Title of the short code"`
		Description  string     `json:"description,omitempty" doc:"Short code description"`
		Slug         string     `json:"slug" required:"true" doc:"Short URL slug"`
		TargetURL    string     `json:"target_url" required:"true" doc:"Target URL to redirect to"`
		FallbackURL  string     `json:"fallback_url,omitempty" doc:"Fallback URL when link is not valid"`
		IsCrawlable  bool       `json:"is_crawlable,omitempty" doc:"Include in robots.txt"`
		ForwardQuery bool       `json:"forward_query,omitempty" doc:"Forward query string to target"`
		ValidSince   *time.Time `json:"valid_since,omitempty" doc:"Start of validity window"`
		ValidUntil   *time.Time `json:"valid_until,omitempty" doc:"End of validity window"`
		Domains      []string   `json:"domains" required:"true" minItems:"1" doc:"Domain FQDNs where this short code is available"`
		Tags         []string   `json:"tags,omitempty" doc:"Tag slugs to associate with this short code"`
	}
}

// PatchShortCodeInput is the request for partially updating a short code.
type PatchShortCodeInput struct {
	PublicID string `path:"id" doc:"Public ID of the short code"`
	Body     struct {
		Title        *string    `json:"title,omitempty" doc:"Title of the short code"`
		Description  *string    `json:"description,omitempty" doc:"Short code description"`
		Slug         *string    `json:"slug,omitempty" doc:"Short URL slug"`
		TargetURL    *string    `json:"target_url,omitempty" doc:"Target URL to redirect to"`
		FallbackURL  *string    `json:"fallback_url,omitempty" doc:"Fallback URL when link is not valid"`
		IsCrawlable  *bool      `json:"is_crawlable,omitempty" doc:"Include in robots.txt"`
		ForwardQuery *bool      `json:"forward_query,omitempty" doc:"Forward query string to target"`
		ValidSince   *time.Time `json:"valid_since,omitempty" doc:"Start of validity window"`
		ValidUntil   *time.Time `json:"valid_until,omitempty" doc:"End of validity window"`
		Domains      *[]string  `json:"domains,omitempty" minItems:"1" doc:"Domain FQDNs where this short code is available"`
		Tags         *[]string  `json:"tags,omitempty" doc:"Tag slugs to associate with this short code"`
	}
}

// DeleteShortCodeInput is the request for deleting a short code.
type DeleteShortCodeInput struct {
	PublicID string `path:"id" doc:"Public ID of the short code"`
}

// ListShortCodesOutput is the response for listing short codes.
type ListShortCodesOutput struct {
	Body []ShortCodeBody
}

func shortCodeDataToBody(d *ShortCodeData) ShortCodeBody {
	domains := d.Domains
	if domains == nil {
		domains = []string{}
	}
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	visitsByDomain := d.VisitsByDomain
	if visitsByDomain == nil {
		visitsByDomain = map[string]int{}
	}
	visitsByCampaign := d.VisitsByCampaign
	if visitsByCampaign == nil {
		visitsByCampaign = map[string]int{}
	}
	visitsByCountry := d.VisitsByCountry
	if visitsByCountry == nil {
		visitsByCountry = map[string]int{}
	}
	return ShortCodeBody{
		PublicID:         d.PublicID,
		Title:            d.Title,
		Description:      d.Description,
		Slug:             d.Slug,
		TargetURL:        d.TargetURL,
		FallbackURL:      d.FallbackURL,
		IsCrawlable:      d.IsCrawlable,
		ForwardQuery:     d.ForwardQuery,
		ValidSince:       d.ValidSince,
		ValidUntil:       d.ValidUntil,
		Domains:          domains,
		Tags:             tags,
		Visits:           d.Visits,
		VisitsThisWeek:   d.VisitsThisWeek,
		VisitsByDomain:   visitsByDomain,
		VisitsByCampaign: visitsByCampaign,
		VisitsByCountry:  visitsByCountry,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

// shortCodeError maps store errors to HTTP errors: conflicts become 409,
// missing resources 404, and anything else an internal error with the given
// fallback message.
func shortCodeError(err error, fallback string) error {
	switch {
	case errors.Is(err, ErrConflict):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, ErrNotFound):
		return huma.Error404NotFound(err.Error())
	default:
		return huma.Error500InternalServerError(fallback, err)
	}
}

// RegisterShortCodeRoutes registers all short code CRUD operations on the given Huma API.
func RegisterShortCodeRoutes(api huma.API, store ShortCodeStore) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-short-code",
		Method:        http.MethodPost,
		Path:          "/api/short-codes",
		Summary:       "Create a short code",
		Tags:          []string{"Short Codes"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, input *CreateShortCodeInput) (*ShortCodeOutput, error) {
		data := &ShortCodeData{
			Title:        input.Body.Title,
			Description:  input.Body.Description,
			Slug:         input.Body.Slug,
			TargetURL:    input.Body.TargetURL,
			FallbackURL:  input.Body.FallbackURL,
			IsCrawlable:  input.Body.IsCrawlable,
			ForwardQuery: input.Body.ForwardQuery,
			ValidSince:   input.Body.ValidSince,
			ValidUntil:   input.Body.ValidUntil,
			Domains:      input.Body.Domains,
			Tags:         input.Body.Tags,
		}
		created, err := store.CreateShortCode(ctx, data)
		if err != nil {
			return nil, shortCodeError(err, "failed to create short code")
		}
		return &ShortCodeOutput{Body: shortCodeDataToBody(created)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-short-code",
		Method:      http.MethodGet,
		Path:        "/api/short-codes/{id}",
		Summary:     "Get a short code",
		Tags:        []string{"Short Codes"},
	}, func(ctx context.Context, input *GetShortCodeInput) (*ShortCodeOutput, error) {
		data, err := store.GetShortCode(ctx, input.PublicID)
		if err != nil {
			return nil, shortCodeError(err, "failed to load short code")
		}
		return &ShortCodeOutput{Body: shortCodeDataToBody(data)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-short-codes",
		Method:      http.MethodGet,
		Path:        "/api/short-codes",
		Summary:     "List all short codes",
		Tags:        []string{"Short Codes"},
	}, func(ctx context.Context, input *struct{}) (*ListShortCodesOutput, error) {
		codes, err := store.ListShortCodes(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list short codes", err)
		}
		bodies := make([]ShortCodeBody, len(codes))
		for i, c := range codes {
			bodies[i] = shortCodeDataToBody(c)
		}
		return &ListShortCodesOutput{Body: bodies}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-short-code",
		Method:      http.MethodPut,
		Path:        "/api/short-codes/{id}",
		Summary:     "Update a short code",
		Tags:        []string{"Short Codes"},
	}, func(ctx context.Context, input *UpdateShortCodeInput) (*ShortCodeOutput, error) {
		data := &ShortCodeData{
			Title:        input.Body.Title,
			Description:  input.Body.Description,
			Slug:         input.Body.Slug,
			TargetURL:    input.Body.TargetURL,
			FallbackURL:  input.Body.FallbackURL,
			IsCrawlable:  input.Body.IsCrawlable,
			ForwardQuery: input.Body.ForwardQuery,
			ValidSince:   input.Body.ValidSince,
			ValidUntil:   input.Body.ValidUntil,
			Domains:      input.Body.Domains,
			Tags:         input.Body.Tags,
		}
		updated, err := store.UpdateShortCode(ctx, input.PublicID, data)
		if err != nil {
			return nil, shortCodeError(err, "failed to load short code")
		}
		return &ShortCodeOutput{Body: shortCodeDataToBody(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "patch-short-code",
		Method:      http.MethodPatch,
		Path:        "/api/short-codes/{id}",
		Summary:     "Partially update a short code",
		Tags:        []string{"Short Codes"},
	}, func(ctx context.Context, input *PatchShortCodeInput) (*ShortCodeOutput, error) {
		patch := &ShortCodePatch{
			Title:        input.Body.Title,
			Description:  input.Body.Description,
			Slug:         input.Body.Slug,
			TargetURL:    input.Body.TargetURL,
			FallbackURL:  input.Body.FallbackURL,
			IsCrawlable:  input.Body.IsCrawlable,
			ForwardQuery: input.Body.ForwardQuery,
			ValidSince:   input.Body.ValidSince,
			ValidUntil:   input.Body.ValidUntil,
			Domains:      input.Body.Domains,
			Tags:         input.Body.Tags,
		}
		updated, err := store.PatchShortCode(ctx, input.PublicID, patch)
		if err != nil {
			return nil, shortCodeError(err, "failed to load short code")
		}
		return &ShortCodeOutput{Body: shortCodeDataToBody(updated)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-short-code",
		Method:        http.MethodDelete,
		Path:          "/api/short-codes/{id}",
		Summary:       "Delete a short code",
		Tags:          []string{"Short Codes"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, input *DeleteShortCodeInput) (*struct{}, error) {
		err := store.DeleteShortCode(ctx, input.PublicID)
		if err != nil {
			return nil, shortCodeError(err, "failed to load short code")
		}
		return nil, nil
	})
}
