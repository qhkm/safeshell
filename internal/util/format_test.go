package util

import (
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 512, "512 B"},
		{"exactly 1KB", 1024, "1.0 KB"},
		{"kilobytes", 2048, "2.0 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"megabytes with decimals", 1572864, "1.5 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"large gigabytes", 5368709120, "5.0 GB"},
		{"terabytes", 1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"1 minute ago", now.Add(-1 * time.Minute), "1 minute ago"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1 hour ago"},
		{"3 hours ago", now.Add(-3 * time.Hour), "3 hours ago"},
		{"1 day ago", now.Add(-24 * time.Hour), "1 day ago"},
		{"3 days ago", now.Add(-72 * time.Hour), "3 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("FormatTimeAgo() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgoOldDate(t *testing.T) {
	// For dates older than 7 days, it should return formatted date
	oldTime := time.Now().Add(-14 * 24 * time.Hour)
	result := FormatTimeAgo(oldTime)

	// Should be in format "Jan 2, 15:04"
	expected := oldTime.Format("Jan 2, 15:04")
	if result != expected {
		t.Errorf("FormatTimeAgo() for old date = %q, want %q", result, expected)
	}
}

// Benchmarks
func BenchmarkFormatBytes(b *testing.B) {
	sizes := []int64{0, 512, 1024, 1048576, 1073741824}
	for i := 0; i < b.N; i++ {
		for _, size := range sizes {
			FormatBytes(size)
		}
	}
}

func BenchmarkFormatTimeAgo(b *testing.B) {
	times := []time.Time{
		time.Now().Add(-30 * time.Second),
		time.Now().Add(-5 * time.Minute),
		time.Now().Add(-3 * time.Hour),
		time.Now().Add(-2 * 24 * time.Hour),
	}
	for i := 0; i < b.N; i++ {
		for _, t := range times {
			FormatTimeAgo(t)
		}
	}
}
