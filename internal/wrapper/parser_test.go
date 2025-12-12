package wrapper

import (
	"reflect"
	"testing"
)

func TestParseRmArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "single file",
			args:     []string{"file.txt"},
			expected: []string{"file.txt"},
		},
		{
			name:     "multiple files",
			args:     []string{"file1.txt", "file2.txt", "file3.txt"},
			expected: []string{"file1.txt", "file2.txt", "file3.txt"},
		},
		{
			name:     "with -f flag",
			args:     []string{"-f", "file.txt"},
			expected: []string{"file.txt"},
		},
		{
			name:     "with -rf flags",
			args:     []string{"-rf", "directory"},
			expected: []string{"directory"},
		},
		{
			name:     "with separated flags",
			args:     []string{"-r", "-f", "dir1", "dir2"},
			expected: []string{"dir1", "dir2"},
		},
		{
			name:     "with --recursive flag",
			args:     []string{"--recursive", "directory"},
			expected: []string{"directory"},
		},
		{
			name:     "flags only",
			args:     []string{"-rf"},
			expected: []string{},
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRmArgs(tt.args)
			if err != nil {
				t.Fatalf("ParseRmArgs returned error: %v", err)
			}

			if result == nil {
				result = []string{}
			}
			if tt.expected == nil {
				tt.expected = []string{}
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseRmArgs(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestParseMvArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "source and dest",
			args:     []string{"source.txt", "dest.txt"},
			expected: []string{"source.txt"},
		},
		{
			name:     "multiple sources to directory",
			args:     []string{"file1.txt", "file2.txt", "dir/"},
			expected: []string{"file1.txt", "file2.txt"},
		},
		{
			name:     "with -f flag",
			args:     []string{"-f", "source.txt", "dest.txt"},
			expected: []string{"source.txt"},
		},
		{
			name:     "with -n flag",
			args:     []string{"-n", "source.txt", "dest.txt"},
			expected: []string{"source.txt"},
		},
		{
			name:     "single arg (edge case)",
			args:     []string{"file.txt"},
			expected: []string{"file.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseMvArgs(tt.args)
			if err != nil {
				t.Fatalf("ParseMvArgs returned error: %v", err)
			}

			if result == nil {
				result = []string{}
			}
			if tt.expected == nil {
				tt.expected = []string{}
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseMvArgs(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestParseCpArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "source and dest",
			args:     []string{"source.txt", "dest.txt"},
			expected: []string{"dest.txt"},
		},
		{
			name:     "multiple sources to directory",
			args:     []string{"file1.txt", "file2.txt", "dir/"},
			expected: []string{"dir/"},
		},
		{
			name:     "with -r flag",
			args:     []string{"-r", "srcdir", "destdir"},
			expected: []string{"destdir"},
		},
		{
			name:     "single arg",
			args:     []string{"file.txt"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCpArgs(tt.args)
			if err != nil {
				t.Fatalf("ParseCpArgs returned error: %v", err)
			}

			if result == nil {
				result = []string{}
			}
			if tt.expected == nil {
				tt.expected = []string{}
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseCpArgs(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestParseChmodArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "mode and file",
			args:     []string{"755", "script.sh"},
			expected: []string{"script.sh"},
		},
		{
			name:     "mode and multiple files",
			args:     []string{"644", "file1.txt", "file2.txt"},
			expected: []string{"file1.txt", "file2.txt"},
		},
		{
			name:     "with -R flag",
			args:     []string{"-R", "755", "directory"},
			expected: []string{"directory"},
		},
		{
			name:     "symbolic mode",
			args:     []string{"+x", "script.sh"},
			expected: []string{"script.sh"},
		},
		{
			name:     "single arg (just mode)",
			args:     []string{"755"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseChmodArgs(tt.args)
			if err != nil {
				t.Fatalf("ParseChmodArgs returned error: %v", err)
			}

			if result == nil {
				result = []string{}
			}
			if tt.expected == nil {
				tt.expected = []string{}
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseChmodArgs(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestParseChownArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "owner and file",
			args:     []string{"user", "file.txt"},
			expected: []string{"file.txt"},
		},
		{
			name:     "owner:group and file",
			args:     []string{"user:group", "file.txt"},
			expected: []string{"file.txt"},
		},
		{
			name:     "owner and multiple files",
			args:     []string{"user", "file1.txt", "file2.txt"},
			expected: []string{"file1.txt", "file2.txt"},
		},
		{
			name:     "with -R flag",
			args:     []string{"-R", "user:group", "directory"},
			expected: []string{"directory"},
		},
		{
			name:     "single arg (just owner)",
			args:     []string{"user"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseChownArgs(tt.args)
			if err != nil {
				t.Fatalf("ParseChownArgs returned error: %v", err)
			}

			if result == nil {
				result = []string{}
			}
			if tt.expected == nil {
				tt.expected = []string{}
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseChownArgs(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}
