package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/davidolrik/corto/internal/server/handlers"
)

type fakeStatsStore struct {
	data *handlers.StatsData
}

func (s *fakeStatsStore) GetStats(_ context.Context) (*handlers.StatsData, error) {
	return s.data, nil
}

func TestGetStats(t *testing.T) {
	_, api := humatest.New(t)
	handlers.RegisterStatsRoutes(api, &fakeStatsStore{data: &handlers.StatsData{
		Links:           3,
		Domains:         2,
		Tags:            1,
		Visits:          120,
		VisitsThisWeek:  17,
		VisitsByCountry: map[string]int{"DK": 100, "GB": 15, "unknown": 5},
	}})

	resp := api.Get("/api/stats")

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var body handlers.StatsBody
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.Links != 3 {
		t.Errorf("expected 3 links, got %d", body.Links)
	}
	if body.Domains != 2 {
		t.Errorf("expected 2 domains, got %d", body.Domains)
	}
	if body.Tags != 1 {
		t.Errorf("expected 1 tag, got %d", body.Tags)
	}
	if body.Visits != 120 {
		t.Errorf("expected 120 visits, got %d", body.Visits)
	}
	if body.VisitsThisWeek != 17 {
		t.Errorf("expected 17 visits this week, got %d", body.VisitsThisWeek)
	}
	if body.VisitsByCountry["DK"] != 100 {
		t.Errorf("expected 100 visits from DK, got %d", body.VisitsByCountry["DK"])
	}
}
