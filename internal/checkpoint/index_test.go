package checkpoint

import (
	"sort"
	"testing"
	"time"
)

func TestIndexEntrySorting(t *testing.T) {
	now := time.Now()

	entries := []*IndexEntry{
		{ID: "c", Timestamp: now.Add(-1 * time.Hour), Sequence: 2},
		{ID: "a", Timestamp: now.Add(-3 * time.Hour), Sequence: 0},
		{ID: "b", Timestamp: now.Add(-2 * time.Hour), Sequence: 1},
	}

	// Sort ascending by timestamp
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	if entries[0].ID != "a" || entries[1].ID != "b" || entries[2].ID != "c" {
		t.Errorf("Ascending sort failed: got %s, %s, %s", entries[0].ID, entries[1].ID, entries[2].ID)
	}

	// Sort descending by timestamp
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	if entries[0].ID != "c" || entries[1].ID != "b" || entries[2].ID != "a" {
		t.Errorf("Descending sort failed: got %s, %s, %s", entries[0].ID, entries[1].ID, entries[2].ID)
	}
}

func TestIndexEntrySortingWithTiebreaker(t *testing.T) {
	now := time.Now()

	// Same timestamp, different sequences
	entries := []*IndexEntry{
		{ID: "c", Timestamp: now, Sequence: 2},
		{ID: "a", Timestamp: now, Sequence: 0},
		{ID: "b", Timestamp: now, Sequence: 1},
	}

	// Sort descending by sequence as tiebreaker
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Timestamp.Equal(entries[j].Timestamp) {
			return entries[i].Sequence > entries[j].Sequence
		}
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	if entries[0].ID != "c" || entries[1].ID != "b" || entries[2].ID != "a" {
		t.Errorf("Tiebreaker sort failed: got %s, %s, %s", entries[0].ID, entries[1].ID, entries[2].ID)
	}
}

// Benchmarks to show sort.Slice performance vs bubble sort

func generateTestEntries(n int) []*IndexEntry {
	entries := make([]*IndexEntry, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		entries[i] = &IndexEntry{
			ID:        string(rune('a' + (i % 26))),
			Timestamp: now.Add(-time.Duration(n-i) * time.Minute),
			Sequence:  int64(i),
		}
	}
	// Shuffle
	for i := len(entries) - 1; i > 0; i-- {
		j := i / 2
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries
}

func BenchmarkSortSlice10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		entries := generateTestEntries(10)
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Timestamp.Equal(entries[j].Timestamp) {
				return entries[i].Sequence > entries[j].Sequence
			}
			return entries[i].Timestamp.After(entries[j].Timestamp)
		})
	}
}

func BenchmarkSortSlice100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		entries := generateTestEntries(100)
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Timestamp.Equal(entries[j].Timestamp) {
				return entries[i].Sequence > entries[j].Sequence
			}
			return entries[i].Timestamp.After(entries[j].Timestamp)
		})
	}
}

func BenchmarkSortSlice1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		entries := generateTestEntries(1000)
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Timestamp.Equal(entries[j].Timestamp) {
				return entries[i].Sequence > entries[j].Sequence
			}
			return entries[i].Timestamp.After(entries[j].Timestamp)
		})
	}
}

// Bubble sort implementation for benchmark comparison
func bubbleSort(entries []*IndexEntry) {
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			swap := false
			if entries[j].Timestamp.After(entries[i].Timestamp) {
				swap = true
			} else if entries[j].Timestamp.Equal(entries[i].Timestamp) && entries[j].Sequence > entries[i].Sequence {
				swap = true
			}
			if swap {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}

func BenchmarkBubbleSort10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		entries := generateTestEntries(10)
		bubbleSort(entries)
	}
}

func BenchmarkBubbleSort100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		entries := generateTestEntries(100)
		bubbleSort(entries)
	}
}

func BenchmarkBubbleSort1000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		entries := generateTestEntries(1000)
		bubbleSort(entries)
	}
}
