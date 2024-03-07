package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewConfigCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Handle Corto config",
		Long:  "Handle Corto config",
	}

	showCommand := &cobra.Command{
		Use:   "show",
		Short: "Show resolved Corto config (flags > env > config > defaults)",
		Long:  "Show Corto config",
		Run: func(cmd *cobra.Command, args []string) {
			json, _ := json.MarshalIndent(viper.AllSettings(), "", "  ")
			fmt.Println(string(json))
		},
	}
	configCmd.AddCommand(showCommand)

	writeCmd := &cobra.Command{
		Use:     "write",
		Aliases: []string{"init"},
		Short:   "Write config file",
		Long:    "Write config file",
		Run: func(cmd *cobra.Command, args []string) {
			if viper.ConfigFileUsed() != "" {
				slog.Info("Loading config", "config_file", viper.ConfigFileUsed())
			} else {
				slog.Warn("No existing config file, writing a new one with current environment and options")
			}
			configFile := path.Join(viper.GetString("config.path"), "config.toml")
			slog.Info("Writing config", "config_file", configFile)
			err := viper.WriteConfigAs(configFile)
			cobra.CheckErr(err)
		},
	}
	configCmd.AddCommand(writeCmd)

	return configCmd
}
