package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// StatsStore defines the interface for tenant-level statistics.
type StatsStore interface {
	GetStats(ctx context.Context) (*StatsData, error)
}

// StatsData represents tenant-level statistics in the service layer.
type StatsData struct {
	Links           int
	Domains         int
	Tags            int
	Visits          int
	VisitsThisWeek  int
	VisitsByCountry map[string]int
}

// StatsOutput is the response for tenant statistics.
type StatsOutput struct {
	Body StatsBody
}

// StatsBody is the JSON body of a stats response.
type StatsBody struct {
	Links           int            `json:"links" doc:"Number of short links"`
	Domains         int            `json:"domains" doc:"Number of domains"`
	Tags            int            `json:"tags" doc:"Number of tags"`
	Visits          int            `json:"visits" doc:"Total number of recorded visits"`
	VisitsThisWeek  int            `json:"visits_this_week" doc:"Visits recorded in the last 7 days"`
	VisitsByCountry map[string]int `json:"visits_by_country" doc:"Recorded visits per ISO country code; unresolved countries count as \"unknown\""`
}

// RegisterStatsRoutes registers the tenant statistics endpoint on the given Huma API.
func RegisterStatsRoutes(api huma.API, store StatsStore) {
	huma.Register(api, huma.Operation{
		OperationID: "get-stats",
		Method:      http.MethodGet,
		Path:        "/api/stats",
		Summary:     "Get tenant statistics",
		Tags:        []string{"Stats"},
	}, func(ctx context.Context, input *struct{}) (*StatsOutput, error) {
		data, err := store.GetStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load stats", err)
		}
		visitsByCountry := data.VisitsByCountry
		if visitsByCountry == nil {
			visitsByCountry = map[string]int{}
		}
		return &StatsOutput{Body: StatsBody{
			Links:           data.Links,
			Domains:         data.Domains,
			Tags:            data.Tags,
			Visits:          data.Visits,
			VisitsThisWeek:  data.VisitsThisWeek,
			VisitsByCountry: visitsByCountry,
		}}, nil
	})
}
