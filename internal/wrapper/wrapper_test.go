package wrapper

import (
	"testing"
)

func TestIsSupported(t *testing.T) {
	supportedCommands := []string{"rm", "mv", "cp", "chmod", "chown"}
	unsupportedCommands := []string{"ls", "cat", "echo", "grep", "find"}

	for _, cmd := range supportedCommands {
		if !IsSupported(cmd) {
			t.Errorf("Command '%s' should be supported", cmd)
		}
	}

	for _, cmd := range unsupportedCommands {
		if IsSupported(cmd) {
			t.Errorf("Command '%s' should not be supported", cmd)
		}
	}
}

func TestGetCommand(t *testing.T) {
	// Test getting existing command
	cmd, ok := GetCommand("rm")
	if !ok {
		t.Error("Should find 'rm' command")
	}
	if cmd.Name != "rm" {
		t.Errorf("Expected name 'rm', got '%s'", cmd.Name)
	}
	if cmd.RiskLevel != "HIGH" {
		t.Errorf("Expected risk level 'HIGH', got '%s'", cmd.RiskLevel)
	}
	if cmd.Parser == nil {
		t.Error("Parser should not be nil")
	}

	// Test getting non-existent command
	_, ok = GetCommand("nonexistent")
	if ok {
		t.Error("Should not find 'nonexistent' command")
	}
}

func TestSupportedCommandsRiskLevels(t *testing.T) {
	expectedRisks := map[string]string{
		"rm":    "HIGH",
		"mv":    "MEDIUM",
		"cp":    "LOW",
		"chmod": "MEDIUM",
		"chown": "MEDIUM",
	}

	for cmd, expectedRisk := range expectedRisks {
		def, ok := GetCommand(cmd)
		if !ok {
			t.Errorf("Command '%s' should exist", cmd)
			continue
		}
		if def.RiskLevel != expectedRisk {
			t.Errorf("Command '%s' should have risk '%s', got '%s'", cmd, expectedRisk, def.RiskLevel)
		}
	}
}

func TestCommandParsers(t *testing.T) {
	// Test that all supported commands have working parsers
	for name, def := range SupportedCommands {
		if def.Parser == nil {
			t.Errorf("Command '%s' has nil parser", name)
			continue
		}

		// Test with empty args (should not panic)
		_, err := def.Parser([]string{})
		if err != nil {
			t.Errorf("Command '%s' parser failed on empty args: %v", name, err)
		}

		// Test with single arg
		_, err = def.Parser([]string{"file.txt"})
		if err != nil {
			t.Errorf("Command '%s' parser failed on single arg: %v", name, err)
		}
	}
}
