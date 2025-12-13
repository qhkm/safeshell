package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config [get|set] [key] [value]",
	Short: "View or modify SafeShell configuration",
	Long: `View or modify SafeShell configuration settings.

Without arguments, shows all current settings.
Use 'get' to retrieve a specific setting.
Use 'set' to modify a setting.

Available settings:
  retention_days       Days before 'safeshell clean' removes checkpoints (default: 7)
  max_checkpoints      Maximum number of checkpoints to keep (default: 100)
  max_storage_mb       Total storage limit in MB (default: 5000)
  max_file_size_mb     Skip files larger than this in MB (default: 100)
  warn_sensitive_files Warn when backing up sensitive files (default: true)

Examples:
  safeshell config                          # Show all settings
  safeshell config get retention_days       # Get single value
  safeshell config set retention_days 3     # Set to 3 days
  safeshell config set max_storage_mb 2000  # Set storage limit to 2GB`,
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

// configKeys defines valid config keys with descriptions
var configKeys = map[string]string{
	"retention_days":       "Days before cleanup removes checkpoints",
	"max_checkpoints":      "Maximum number of checkpoints to keep",
	"max_storage_mb":       "Total storage limit in MB",
	"max_file_size_mb":     "Skip files larger than this (MB)",
	"warn_sensitive_files": "Warn when backing up sensitive files",
	"safeshell_dir":        "SafeShell data directory",
}

func runConfig(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Show all config
		return showAllConfig()
	}

	action := args[0]

	switch action {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: safeshell config get <key>")
		}
		return getConfig(args[1])

	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: safeshell config set <key> <value>")
		}
		return setConfig(args[1], args[2])

	default:
		// Treat as 'get' if it looks like a key
		if _, ok := configKeys[action]; ok {
			return getConfig(action)
		}
		return fmt.Errorf("unknown action: %s (use 'get' or 'set')", action)
	}
}

func showAllConfig() error {
	fmt.Println("SafeShell Configuration")
	fmt.Println(strings.Repeat("─", 50))

	bold := color.New(color.Bold)

	// Storage settings
	bold.Println("\nStorage:")
	fmt.Printf("  max_storage_mb:       %v\n", viper.Get("max_storage_mb"))
	fmt.Printf("  max_file_size_mb:     %v\n", viper.Get("max_file_size_mb"))
	fmt.Printf("  max_checkpoints:      %v\n", viper.Get("max_checkpoints"))

	// Cleanup settings
	bold.Println("\nCleanup:")
	fmt.Printf("  retention_days:       %v\n", viper.Get("retention_days"))

	// Security settings
	bold.Println("\nSecurity:")
	fmt.Printf("  warn_sensitive_files: %v\n", viper.Get("warn_sensitive_files"))

	// Paths
	bold.Println("\nPaths:")
	fmt.Printf("  safeshell_dir:        %v\n", viper.Get("safeshell_dir"))

	// Exclusions
	excludes := viper.GetStringSlice("exclude_paths")
	if len(excludes) > 0 {
		bold.Println("\nExclude patterns:")
		for _, e := range excludes {
			fmt.Printf("  - %s\n", e)
		}
	}

	// Wrapped commands
	wrapped := viper.GetStringSlice("wrapped_commands")
	if len(wrapped) > 0 {
		bold.Println("\nWrapped commands:")
		fmt.Printf("  %s\n", strings.Join(wrapped, ", "))
	}

	fmt.Println()
	color.HiBlack("Config file: %s/config.yaml", viper.Get("safeshell_dir"))

	return nil
}

func getConfig(key string) error {
	// Validate key
	if _, ok := configKeys[key]; !ok {
		return fmt.Errorf("unknown config key: %s\n\nValid keys: %s",
			key, strings.Join(getValidKeys(), ", "))
	}

	value := viper.Get(key)
	fmt.Printf("%v\n", value)
	return nil
}

func setConfig(key, value string) error {
	// Validate key
	desc, ok := configKeys[key]
	if !ok {
		return fmt.Errorf("unknown config key: %s\n\nValid keys: %s",
			key, strings.Join(getValidKeys(), ", "))
	}

	// Don't allow changing safeshell_dir
	if key == "safeshell_dir" {
		return fmt.Errorf("safeshell_dir cannot be changed after initialization")
	}

	// Parse and validate value based on key type
	var parsedValue interface{}
	var err error

	switch key {
	case "retention_days", "max_checkpoints", "max_storage_mb", "max_file_size_mb":
		parsedValue, err = strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%s must be a number", key)
		}
		if parsedValue.(int) < 0 {
			return fmt.Errorf("%s must be non-negative", key)
		}

	case "warn_sensitive_files":
		lower := strings.ToLower(value)
		if lower == "true" || lower == "1" || lower == "yes" {
			parsedValue = true
		} else if lower == "false" || lower == "0" || lower == "no" {
			parsedValue = false
		} else {
			return fmt.Errorf("%s must be true or false", key)
		}

	default:
		parsedValue = value
	}

	// Set and save
	viper.Set(key, parsedValue)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	color.Green("✓ Set %s = %v", key, parsedValue)
	color.HiBlack("  %s", desc)

	return nil
}

func getValidKeys() []string {
	keys := make([]string, 0, len(configKeys))
	for k := range configKeys {
		keys = append(keys, k)
	}
	return keys
}
