package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	SafeShellDir    string   `mapstructure:"safeshell_dir"`
	RetentionDays   int      `mapstructure:"retention_days"`
	MaxCheckpoints  int      `mapstructure:"max_checkpoints"`
	ExcludePaths    []string `mapstructure:"exclude_paths"`
	WrappedCommands []string `mapstructure:"wrapped_commands"`
}

var cfg *Config

func Init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	safeshellDir := filepath.Join(homeDir, ".safeshell")

	// Create safeshell directory if it doesn't exist
	if err := os.MkdirAll(safeshellDir, 0755); err != nil {
		return err
	}

	// Create checkpoints directory
	checkpointsDir := filepath.Join(safeshellDir, "checkpoints")
	if err := os.MkdirAll(checkpointsDir, 0755); err != nil {
		return err
	}

	viper.SetDefault("safeshell_dir", safeshellDir)
	viper.SetDefault("retention_days", 7)
	viper.SetDefault("max_checkpoints", 100)
	viper.SetDefault("exclude_paths", []string{
		"*.tmp",
		"*.swp",
		"*~",
		".git/objects/*",
		"node_modules/*",
	})
	viper.SetDefault("wrapped_commands", []string{"rm", "mv", "cp", "chmod", "chown"})

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(safeshellDir)

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		// Config file not found, create default one
		if err := viper.SafeWriteConfigAs(filepath.Join(safeshellDir, "config.yaml")); err != nil {
			// Ignore if file already exists
			if _, ok := err.(viper.ConfigFileAlreadyExistsError); !ok {
				return err
			}
		}
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return err
	}

	return nil
}

func Get() *Config {
	if cfg == nil {
		Init()
	}
	return cfg
}

func GetSafeShellDir() string {
	return Get().SafeShellDir
}

func GetCheckpointsDir() string {
	return filepath.Join(GetSafeShellDir(), "checkpoints")
}

func GetOperationsLog() string {
	return filepath.Join(GetSafeShellDir(), "operations.log")
}
