package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/davidolrik/corto/internal/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "corto",
		Short:   "Corto - Url shortener",
		Long:    `Corto - Url shortener`,
		Version: core.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize config and bind global flags to the config
			messages, err := core.InitializeConfig(cmd)
			logger := core.InitializeLogger()
			for _, message := range messages {
				logger.Log(cmd.Context(), message.Level, message.Message)
			}

			return err
		},
	}

	// Global flags
	var configPath string
	rootCmd.PersistentFlags().StringVar(
		&configPath, "config-path", core.UserConfigPath(), "Config path",
	)
	var forceLogToStdout bool
	rootCmd.PersistentFlags().BoolVar(&forceLogToStdout, "log-force-stdout", false, "Force logger to use stdout")
	var debug int
	rootCmd.PersistentFlags().CountVarP(&debug, "debug", "d", "Set debug level")

	debugCmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug",
		Long:  "Debug",
		Run: func(cmd *cobra.Command, args []string) {
			json, _ := json.MarshalIndent(viper.AllSettings(), "", "  ")
			fmt.Println(string(json))
		},
	}
	rootCmd.AddCommand(debugCmd)

	rootCmd.AddCommand(NewConfigCommand())
	rootCmd.AddCommand(NewMigrationCommand())
	rootCmd.AddCommand(NewServerCommand())
	rootCmd.AddCommand(NewUserCommand())
	rootCmd.AddCommand(NewTenantCommand())
	rootCmd.AddCommand(NewImportCommand())

	return rootCmd
}
