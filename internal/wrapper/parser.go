package wrapper

import (
	"strings"
)

// ParseRmArgs parses rm command arguments and returns target paths
func ParseRmArgs(args []string) ([]string, error) {
	var targets []string

	for _, arg := range args {
		// Skip flags
		if strings.HasPrefix(arg, "-") {
			continue
		}
		targets = append(targets, arg)
	}

	return targets, nil
}

// ParseMvArgs parses mv command arguments and returns source paths to backup
func ParseMvArgs(args []string) ([]string, error) {
	var nonFlags []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		nonFlags = append(nonFlags, arg)
	}

	// mv source... dest
	// Backup all sources (they will be moved/deleted)
	if len(nonFlags) >= 2 {
		return nonFlags[:len(nonFlags)-1], nil
	}

	return nonFlags, nil
}

// ParseCpArgs parses cp command arguments and returns destination to backup
func ParseCpArgs(args []string) ([]string, error) {
	var nonFlags []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		nonFlags = append(nonFlags, arg)
	}

	// cp source... dest
	// Backup destination if it exists (might be overwritten)
	if len(nonFlags) >= 2 {
		return []string{nonFlags[len(nonFlags)-1]}, nil
	}

	return []string{}, nil
}

// ParseChmodArgs parses chmod arguments and returns target paths
func ParseChmodArgs(args []string) ([]string, error) {
	var nonFlags []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		nonFlags = append(nonFlags, arg)
	}

	// chmod mode file...
	// Skip the mode argument, return file paths
	if len(nonFlags) >= 2 {
		return nonFlags[1:], nil
	}

	return []string{}, nil
}

// ParseChownArgs parses chown arguments and returns target paths
func ParseChownArgs(args []string) ([]string, error) {
	var nonFlags []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		nonFlags = append(nonFlags, arg)
	}

	// chown owner[:group] file...
	// Skip the owner argument, return file paths
	if len(nonFlags) >= 2 {
		return nonFlags[1:], nil
	}

	return []string{}, nil
}
