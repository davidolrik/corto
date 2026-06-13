package cmd

import (
	"fmt"
	"log/slog"

	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/services"
	"github.com/spf13/cobra"
)

func NewImportCommand() *cobra.Command {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import data from other link shorteners",
		Long:  "Import data from other link shorteners",
	}

	var baseURL, apiKey, tenant, domain string
	var withVisits bool
	shlinkCmd := &cobra.Command{
		Use:   "shlink",
		Short: "Import short links from a Shlink instance",
		Long: `Import short links from a Shlink instance via its REST API.

Domains and tags are created as needed in the given tenant. Short links whose
slug is already taken on their domain are skipped. With --with-visits the
visit history (dates, referers, user agents, countries) is imported as well.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db := core.NewDatabase()
			defer db.Close()

			importer := services.NewShlinkImporter(slog.Default(), db)
			summary, err := importer.Import(cmd.Context(), services.ShlinkImportOptions{
				BaseURL:       baseURL,
				APIKey:        apiKey,
				TenantSlug:    tenant,
				DefaultDomain: domain,
				WithVisits:    withVisits,
			})
			if err != nil {
				return err
			}

			fmt.Printf(
				"Imported %d links (%d merged onto existing links, %d unchanged, %d skipped), %d new domains, %d new tags, %d visits\n",
				summary.ShortCodes, summary.Merged, summary.Unchanged, summary.Skipped, summary.Domains, summary.Tags, summary.Visits,
			)
			return nil
		},
	}
	shlinkCmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL of the Shlink instance")
	shlinkCmd.Flags().StringVar(&apiKey, "api-key", "", "Shlink API key")
	shlinkCmd.Flags().StringVar(&tenant, "tenant", "", "Slug of the corto tenant to import into")
	shlinkCmd.Flags().StringVar(&domain, "domain", "", "Override the corto domain for links on Shlink's default domain (detected from Shlink when omitted)")
	shlinkCmd.Flags().BoolVar(&withVisits, "with-visits", false, "Also import the visit history")
	shlinkCmd.MarkFlagRequired("base-url")
	shlinkCmd.MarkFlagRequired("api-key")
	shlinkCmd.MarkFlagRequired("tenant")
	importCmd.AddCommand(shlinkCmd)

	return importCmd
}
