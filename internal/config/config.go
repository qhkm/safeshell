package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	SafeShellDir       string   `mapstructure:"safeshell_dir"`
	RetentionDays      int      `mapstructure:"retention_days"`
	MaxCheckpoints     int      `mapstructure:"max_checkpoints"`
	MaxStorageMB       int      `mapstructure:"max_storage_mb"`
	MaxFileSizeMB      int      `mapstructure:"max_file_size_mb"`
	WarnSensitiveFiles bool     `mapstructure:"warn_sensitive_files"`
	ExcludePaths       []string `mapstructure:"exclude_paths"`
	SensitivePatterns  []string `mapstructure:"sensitive_patterns"`
	WrappedCommands    []string `mapstructure:"wrapped_commands"`
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
	viper.SetDefault("max_storage_mb", 5000)       // 5GB total storage limit
	viper.SetDefault("max_file_size_mb", 100)      // 100MB per file limit
	viper.SetDefault("warn_sensitive_files", true) // Warn about sensitive files
	viper.SetDefault("exclude_paths", []string{
		"*.tmp",
		"*.swp",
		"*~",
		".git/objects/*",
		"node_modules/*",
	})
	viper.SetDefault("sensitive_patterns", []string{
		".env",
		".env.*",
		"*.pem",
		"*.key",
		"*.p12",
		"*.pfx",
		"id_rsa",
		"id_ed25519",
		"id_ecdsa",
		"*.keystore",
		"credentials.json",
		"service-account*.json",
		"*secret*",
		"*password*",
		".netrc",
		".npmrc",
		".pypirc",
		"aws_credentials",
		".aws/credentials",
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
