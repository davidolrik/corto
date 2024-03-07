package cmd

import (
	"fmt"
	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/migrations"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"os"
)

func NewMigrationCommand() *cobra.Command {
	migrationCmd := &cobra.Command{
		Use:     "migration",
		Aliases: []string{"migrate"},
		Short:   "Manage database migrations",
		Long:    `Manage database migrations for production use. For development, use 'goose' directly.`,
	}

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		Long:  `Apply all pending migrations to bring the database to the latest version.`,
		Run: func(cmd *cobra.Command, args []string) {
			db := core.NewDatabase()
			defer db.Close()

			goose.SetDialect("postgres")
			goose.SetTableName("goose_db_version")
			goose.SetBaseFS(migrations.FS)

			if err := goose.Up(db.DB.DB, "."); err != nil {
				fmt.Fprintf(os.Stderr, "Migration up failed: %s\n", err)
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "Migrations applied successfully\n")
		},
	}
	migrationCmd.AddCommand(upCmd)

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Revert the last migration",
		Long:  `Revert the last applied migration.`,
		Run: func(cmd *cobra.Command, args []string) {
			db := core.NewDatabase()
			defer db.Close()

			goose.SetDialect("postgres")
			goose.SetTableName("goose_db_version")
			goose.SetBaseFS(migrations.FS)

			if err := goose.Down(db.DB.DB, "."); err != nil {
				fmt.Fprintf(os.Stderr, "Migration down failed: %s\n", err)
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "Migration reverted successfully\n")
		},
	}
	migrationCmd.AddCommand(downCmd)

	return migrationCmd
}
