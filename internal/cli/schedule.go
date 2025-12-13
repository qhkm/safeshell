package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage automatic cleanup schedule",
	Long: `Manage automatic cleanup schedule using cron (macOS/Linux).

This command helps you set up automatic checkpoint cleanup.

Examples:
  safeshell schedule                    # Show current schedule
  safeshell schedule enable             # Enable daily cleanup (midnight)
  safeshell schedule enable --hourly    # Enable hourly cleanup
  safeshell schedule enable --keep 10   # Keep 10 most recent checkpoints
  safeshell schedule disable            # Disable automatic cleanup`,
}

var scheduleEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable automatic cleanup",
	RunE:  runScheduleEnable,
}

var scheduleDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable automatic cleanup",
	RunE:  runScheduleDisable,
}

var (
	scheduleHourly  bool
	scheduleKeep    int
	scheduleOlderThan string
)

func init() {
	rootCmd.AddCommand(scheduleCmd)
	scheduleCmd.AddCommand(scheduleEnableCmd)
	scheduleCmd.AddCommand(scheduleDisableCmd)

	scheduleEnableCmd.Flags().BoolVar(&scheduleHourly, "hourly", false, "Run cleanup hourly instead of daily")
	scheduleEnableCmd.Flags().IntVar(&scheduleKeep, "keep", 0, "Keep at least N most recent checkpoints")
	scheduleEnableCmd.Flags().StringVar(&scheduleOlderThan, "older-than", "", "Delete checkpoints older than duration (e.g., 3d, 24h)")

	// Show status when running without subcommand
	scheduleCmd.RunE = runScheduleStatus
}

const cronMarker = "# safeshell-auto-clean"

func runScheduleStatus(cmd *cobra.Command, args []string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("automatic scheduling is not supported on Windows")
	}

	// Get current crontab
	output, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		// No crontab exists
		fmt.Println("Automatic cleanup: disabled")
		fmt.Println()
		fmt.Println("Enable with: safeshell schedule enable")
		return nil
	}

	// Look for our cron job
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, cronMarker) || strings.Contains(line, "safeshell clean") {
			color.Green("Automatic cleanup: enabled")
			fmt.Println()
			fmt.Printf("Schedule: %s\n", describeCronLine(line))
			fmt.Println()
			fmt.Println("Disable with: safeshell schedule disable")
			return nil
		}
	}

	fmt.Println("Automatic cleanup: disabled")
	fmt.Println()
	fmt.Println("Enable with: safeshell schedule enable")
	return nil
}

func runScheduleEnable(cmd *cobra.Command, args []string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("automatic scheduling is not supported on Windows\nUse Task Scheduler manually instead")
	}

	// Get safeshell path
	safeshellPath, err := exec.LookPath("safeshell")
	if err != nil {
		// Try to find our own executable
		safeshellPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("could not find safeshell executable")
		}
	}
	safeshellPath, _ = filepath.Abs(safeshellPath)

	// Build the clean command
	cleanArgs := []string{"clean"}
	if scheduleKeep > 0 {
		cleanArgs = append(cleanArgs, fmt.Sprintf("--keep %d", scheduleKeep))
	}
	if scheduleOlderThan != "" {
		cleanArgs = append(cleanArgs, fmt.Sprintf("--older-than %s", scheduleOlderThan))
	}
	// Default: use retention_days from config (no extra args needed)

	cleanCmd := fmt.Sprintf("%s %s", safeshellPath, strings.Join(cleanArgs, " "))

	// Build cron schedule
	var schedule string
	if scheduleHourly {
		schedule = "0 * * * *" // Every hour at minute 0
	} else {
		schedule = "0 0 * * *" // Daily at midnight
	}

	cronLine := fmt.Sprintf("%s %s %s", schedule, cleanCmd, cronMarker)

	// Get existing crontab
	existingCrontab := ""
	output, err := exec.Command("crontab", "-l").Output()
	if err == nil {
		existingCrontab = string(output)
	}

	// Remove any existing safeshell cron jobs
	var newLines []string
	for _, line := range strings.Split(existingCrontab, "\n") {
		if !strings.Contains(line, cronMarker) && !strings.Contains(line, "safeshell clean") {
			if line != "" || len(newLines) == 0 {
				newLines = append(newLines, line)
			}
		}
	}

	// Add new cron job
	newLines = append(newLines, cronLine)

	// Write new crontab
	newCrontab := strings.Join(newLines, "\n")
	if !strings.HasSuffix(newCrontab, "\n") {
		newCrontab += "\n"
	}

	// Use crontab - to read from stdin
	cmd2 := exec.Command("crontab", "-")
	cmd2.Stdin = strings.NewReader(newCrontab)
	if err := cmd2.Run(); err != nil {
		return fmt.Errorf("failed to update crontab: %w", err)
	}

	color.Green("✓ Automatic cleanup enabled")
	fmt.Println()
	if scheduleHourly {
		fmt.Println("Schedule: Every hour")
	} else {
		fmt.Println("Schedule: Daily at midnight")
	}
	fmt.Printf("Command:  %s\n", cleanCmd)

	return nil
}

func runScheduleDisable(cmd *cobra.Command, args []string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("automatic scheduling is not supported on Windows")
	}

	// Get existing crontab
	output, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		fmt.Println("No scheduled cleanup found.")
		return nil
	}

	// Remove safeshell cron jobs
	var newLines []string
	removed := false
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, cronMarker) || strings.Contains(line, "safeshell clean") {
			removed = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !removed {
		fmt.Println("No scheduled cleanup found.")
		return nil
	}

	// Write new crontab
	newCrontab := strings.Join(newLines, "\n")

	// Handle empty crontab
	newCrontab = strings.TrimSpace(newCrontab)
	if newCrontab == "" {
		// Remove crontab entirely
		exec.Command("crontab", "-r").Run()
	} else {
		if !strings.HasSuffix(newCrontab, "\n") {
			newCrontab += "\n"
		}
		cmd2 := exec.Command("crontab", "-")
		cmd2.Stdin = strings.NewReader(newCrontab)
		if err := cmd2.Run(); err != nil {
			return fmt.Errorf("failed to update crontab: %w", err)
		}
	}

	color.Green("✓ Automatic cleanup disabled")
	return nil
}

func describeCronLine(line string) string {
	parts := strings.Fields(line)
	if len(parts) < 5 {
		return "unknown"
	}

	// Parse cron schedule
	minute, hour := parts[0], parts[1]

	if minute == "0" && hour == "*" {
		return "Every hour"
	}
	if minute == "0" && hour == "0" {
		return "Daily at midnight"
	}
	if hour == "*" {
		return fmt.Sprintf("Every hour at minute %s", minute)
	}
	return fmt.Sprintf("Daily at %s:%s", hour, minute)
}

// promptYesNo asks the user for confirmation
func promptYesNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
