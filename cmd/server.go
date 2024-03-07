package cmd

import (
	"fmt"
	"log/slog"
	"net/http"

	"aidanwoods.dev/go-paseto"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/server/handlers"
	"github.com/davidolrik/corto/internal/server/middlewares"
	"github.com/davidolrik/corto/internal/services"
	"github.com/davidolrik/corto/web"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverAddr builds the listen address from the configured IP and port.
func serverAddr() string {
	return fmt.Sprintf(
		"%s:%d",
		viper.GetString("server.ip"),
		viper.GetInt("server.port"),
	)
}

func NewServerCommand() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:     "server",
		Aliases: []string{},
		Short:   "Manage the Corto API server",
		Long:    `Manage the Corto API server`,
		Run: func(cmd *cobra.Command, args []string) {
			mux := http.NewServeMux()

			config := huma.DefaultConfig("Corto", core.Version)
			config.Info.Description = "A modern and flexible link shortener"
			api := humago.New(mux, config)

			db := core.NewDatabase()
			log := slog.Default()

			authService, err := services.NewAuthService(log, db, viper.GetString("server.private_key"))
			if err != nil {
				slog.Error("Failed to initialize auth service", "error", err)
				return
			}

			// GeoIP is optional; without a database visits are recorded
			// without a country
			var countries services.CountryResolver
			if geoipPath := viper.GetString("geoip.database"); geoipPath != "" {
				geo, err := core.NewGeoIP(geoipPath)
				if err != nil {
					slog.Error("Failed to open GeoIP database", "error", err)
					return
				}
				defer geo.Close()
				countries = geo
			}

			domainService := services.NewDomainService(log, db)
			tagService := services.NewTagService(log, db)
			shortCodeService := services.NewShortCodeService(log, db)
			redirectService := services.NewRedirectService(log, db, countries)
			statsService := services.NewStatsService(log, db)
			userService := services.NewUserService(log, db)

			handlers.RegisterAuthRoutes(api, authService)
			handlers.RegisterDomainRoutes(api, domainService)
			handlers.RegisterTagRoutes(api, tagService)
			handlers.RegisterShortCodeRoutes(api, shortCodeService)
			handlers.RegisterStatsRoutes(api, statsService)
			handlers.RegisterProfileRoutes(api, userService)
			handlers.RegisterVersionRoutes(api, core.Version)
			handlers.RegisterRedirectRoutes(mux, redirectService)
			handlers.RegisterUIRoutes(mux, web.FS())

			publicKey, err := paseto.NewV4AsymmetricPublicKeyFromHex(viper.GetString("server.public_key"))
			if err != nil {
				slog.Error("Failed to parse public key", "error", err)
				return
			}

			// Non-API paths (redirects, docs, OpenAPI spec) are public by
			// default; login and the version endpoint are the public API paths.
			authMiddleware := middlewares.Auth(publicKey, []string{"/api/auth/login", "/api/version"})

			addr := serverAddr()
			slog.Info("Starting server", "addr", addr)
			if err := http.ListenAndServe(addr, authMiddleware(mux)); err != nil {
				slog.Error("Server failed", "error", err)
				cobra.CheckErr(err)
			}
		},
	}

	return serverCmd
}
