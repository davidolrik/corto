package core

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/davidolrik/corto/internal/model"
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bunslog"
)

type Database struct {
	*bun.DB
}

func NewDatabase() Database {
	insecure := !viper.GetBool("database.use_ssl")
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithAddr(fmt.Sprintf(
			"%s:%d",
			viper.GetString("database.host"),
			viper.GetInt("database.port"),
		)),
		pgdriver.WithUser(viper.GetString("database.username")),
		pgdriver.WithPassword(viper.GetString("database.password")),
		pgdriver.WithDatabase(viper.GetString("database.schema")),
		pgdriver.WithInsecure(insecure),
	))

	err := sqldb.Ping()
	if err != nil {
		slog.Error(fmt.Sprintf("Unable to ping database: %v", err))
	}

	db := bun.NewDB(sqldb, pgdialect.New())

	if viper.GetBool("debug.log_sql") {
		db.AddQueryHook(bunslog.NewQueryHook(
			bunslog.WithLogger(slog.Default()),
			// bunslog.WithSlowQueryThreshold(200 * time.Millisecond),
		))
	}

	// Register many-to-many models so bun can better recognize m2m relations.
	db.RegisterModel((*model.TenantUserAccess)(nil))
	db.RegisterModel((*model.ShortCodeTag)(nil))

	return Database{
		DB: db,
	}
}
