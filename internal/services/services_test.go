package services_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/model"
	"github.com/davidolrik/corto/internal/services"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// testDatabase connects to the development database documented in CLAUDE.md.
// Integration tests are skipped when it is not reachable.
func testDatabase(t *testing.T) core.Database {
	t.Helper()

	viper.Set("database.host", "127.0.0.1")
	viper.Set("database.port", 5432)
	viper.Set("database.username", "corto")
	viper.Set("database.password", "corto")
	viper.Set("database.schema", "corto")

	db := core.NewDatabase()
	if err := db.Ping(); err != nil {
		t.Skipf("integration test requires the development database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// createTestUser creates a user with a unique username and registers cleanup.
func createTestUser(t *testing.T, db core.Database, password string) *model.User {
	t.Helper()

	svc := services.NewUserService(testLogger(), db)
	user, err := svc.CreateUser(context.Background(), "user-"+uuid.NewString(), password)
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	t.Cleanup(func() { deleteTestUser(t, db, user.ID) })
	return user
}

func deleteTestUser(t *testing.T, db core.Database, userID int) {
	t.Helper()

	ctx := context.Background()
	if _, err := db.NewDelete().Model((*model.User)(nil)).Where("id = ?", userID).Exec(ctx); err != nil {
		t.Errorf("cleaning up user %d: %v", userID, err)
	}
}

// createTestTenant creates a tenant owned by the given user and registers cleanup.
func createTestTenant(t *testing.T, db core.Database, owner *model.User) *model.Tenant {
	t.Helper()

	svc := services.NewTenantService(testLogger(), db)
	tenant, err := svc.CreateTenant(context.Background(), "Tenant "+uuid.NewString(), "", owner.Username)
	if err != nil {
		t.Fatalf("creating test tenant: %v", err)
	}
	t.Cleanup(func() { deleteTestTenant(t, db, tenant.ID) })
	return tenant
}

func deleteTestTenant(t *testing.T, db core.Database, tenantID int) {
	t.Helper()

	ctx := context.Background()
	if _, err := db.NewDelete().Model((*model.TenantUserAccess)(nil)).Where("tenant_id = ?", tenantID).Exec(ctx); err != nil {
		t.Errorf("cleaning up tenant access %d: %v", tenantID, err)
	}
	if _, err := db.NewDelete().Model((*model.Tenant)(nil)).Where("id = ?", tenantID).Exec(ctx); err != nil {
		t.Errorf("cleaning up tenant %d: %v", tenantID, err)
	}
}
