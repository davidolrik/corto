package cmd

import (
	"fmt"
	"log/slog"

	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/services"
	"github.com/spf13/cobra"
)

func NewTenantCommand() *cobra.Command {
	tenantCmd := &cobra.Command{
		Use:   "tenant",
		Short: "Manage tenants",
		Long:  "Manage tenants",
	}

	var owner string
	var slug string
	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a tenant",
		Long:  "Create a tenant owned by the given user, granting the owner admin access",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db := core.NewDatabase()
			defer db.Close()

			tenantService := services.NewTenantService(slog.Default(), db)
			tenant, err := tenantService.CreateTenant(cmd.Context(), args[0], slug, owner)
			if err != nil {
				return err
			}

			fmt.Printf("Created tenant %q with slug %q and public ID %s\n", tenant.Name, tenant.Slug, tenant.PublicID)
			return nil
		},
	}
	createCmd.Flags().StringVar(&owner, "owner", "", "Username of the owning user")
	createCmd.Flags().StringVar(&slug, "slug", "", "Tenant slug (derived from the name when omitted)")
	createCmd.MarkFlagRequired("owner")
	tenantCmd.AddCommand(createCmd)

	return tenantCmd
}
