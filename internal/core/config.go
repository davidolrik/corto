package core

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"

	"aidanwoods.dev/go-paseto"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func UserConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err) // notest
	}
	return path.Join(home, ".config", "corto")
}

func SystemConfigPath() string {
	return path.Join("etc", "corto")
}

type LogMessage struct {
	Message string
	Level   slog.Level
}

func InitializeConfig(cmd *cobra.Command) ([]LogMessage, error) {
	// Logging is not allowed before after the config has been loaded, so any messages should be appended here
	messages := []LogMessage{}

	// User supplied config path always wins, but it must exist to be used
	configPathFlag := cmd.Flag("config-path")

	// Default to user home directory
	computedUserConfigPath := UserConfigPath()

	// If user has specified a config path override it in the config
	if configPathFlag.Changed {
		computedUserConfigPath = configPathFlag.Value.String()
		viper.Set("config_path", computedUserConfigPath)
	} else
	// User has set config path in the environment
	if userConfigPath, isSet := os.LookupEnv("CORTO_CONFIG_PATH"); isSet {
		fmt.Println("Setting config path from environment")
		computedUserConfigPath = userConfigPath
	}

	_, err := os.Stat(computedUserConfigPath)
	if errors.Is(err, os.ErrNotExist) && computedUserConfigPath == UserConfigPath() {
		os.MkdirAll(computedUserConfigPath, fs.FileMode(0o755))
	} else
	// Abort on any other error
	if err != nil {
		messages = append(messages, LogMessage{"Config path not found", slog.LevelError})
		return messages, err
	}

	viper.AddConfigPath(computedUserConfigPath)
	viper.AddConfigPath(SystemConfigPath())

	// Setup defaults
	viper.SetEnvPrefix("corto")
	// In order to get environment variables mapped into sections, we need to replace . with _
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // read in environment variables that match
	viper.SetConfigName("config")
	viper.SetConfigType("toml")

	viper.SetDefault("debug", 0)
	viper.SetDefault("environment", "development")

	viper.SetDefault("config.path", UserConfigPath())

	viper.SetDefault("server.ip", "127.0.0.1")
	viper.SetDefault("server.port", 3000)
	viper.SetDefault("server.use_ssl", false)

	viper.SetDefault("database.host", "127.0.0.1")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.schema", "corto")
	viper.SetDefault("database.username", "corto")
	viper.SetDefault("database.password", "corto")
	viper.SetDefault("database.use_ssl", false)
	viper.SetDefault("database.log_sql", false)

	viper.SetDefault("log.force_stdout", false)

	// Path to a MaxMind format country database; empty disables GeoIP lookups
	viper.SetDefault("geoip.database", "")


	// Random secret keys as default, only useful for testing
	secretKey := paseto.NewV4AsymmetricSecretKey()
	publicKey := secretKey.Public()
	viper.SetDefault("server.private_key", secretKey.ExportHex())
	viper.SetDefault("server.public_key", publicKey.ExportHex())

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, running on defaults
			messages = append(messages, LogMessage{"No config file found, using defaults", slog.LevelInfo})
		} else {
			// Config file was found but another error was produced
			messages = append(messages, LogMessage{"Unable to parse config", slog.LevelError})
			return messages, err
		}
	}

	// Bind the current command's flags to viper
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Determine the naming convention of the flags when represented in the config file
		configName := f.Name
		configName = strings.Replace(configName, "-", ".", 1)
		configName = strings.ReplaceAll(configName, "-", "_")

		// If flag is set, update viper from flag
		if f.Changed {
			viper.Set(configName, f.Value.String())
		} else
		// Flag is not set, update flag from viper
		if !f.Changed && viper.IsSet(configName) {
			val := viper.Get(configName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})

	return messages, nil
}
