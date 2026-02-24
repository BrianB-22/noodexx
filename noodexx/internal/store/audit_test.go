package store

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestAddAuditEntry(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test-audit-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Initialize store
	store, err := NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		opType  string
		details string
		userCtx string
		wantErr bool
	}{
		{
			name:    "add ingest entry",
			opType:  "ingest",
			details: "Ingested document.txt (1024 bytes)",
			userCtx: "user-session-123",
			wantErr: false,
		},
		{
			name:    "add query entry",
			opType:  "query",
			details: "Query: What is the meaning of life?",
			userCtx: "session-456",
			wantErr: false,
		},
		{
			name:    "add delete entry",
			opType:  "delete",
			details: "Deleted source: old-document.pdf",
			userCtx: "admin",
			wantErr: false,
		},
		{
			name:    "add config entry",
			opType:  "config",
			details: "Changed provider from ollama to openai",
			userCtx: "system",
			wantErr: false,
		},
		{
			name:    "empty details",
			opType:  "ingest",
			details: "",
			userCtx: "user",
			wantErr: false,
		},
		{
			name:    "empty user context",
			opType:  "query",
			details: "Some query",
			userCtx: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.AddAuditEntry(ctx, tt.opType, tt.details, tt.userCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddAuditEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAuditLog(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test-audit-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Initialize store
	store, err := NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add test entries
	entries := []struct {
		opType  string
		details string
		userCtx string
	}{
		{"ingest", "Ingested doc1.txt", "user1"},
		{"query", "Query about AI", "session1"},
		{"delete", "Deleted doc2.pdf", "user1"},
		{"ingest", "Ingested doc3.md", "user2"},
		{"config", "Changed settings", "admin"},
	}

	for _, entry := range entries {
		err := store.AddAuditEntry(ctx, entry.opType, entry.details, entry.userCtx)
		if err != nil {
			t.Fatalf("Failed to add audit entry: %v", err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Get the earliest and latest timestamps for date range tests
	allEntries, err := store.GetAuditLog(ctx, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("Failed to get all entries: %v", err)
	}
	if len(allEntries) == 0 {
		t.Fatal("No entries found")
	}

	// Find earliest and latest timestamps
	earliest := allEntries[len(allEntries)-1].Timestamp // Last in DESC order
	latest := allEntries[0].Timestamp                   // First in DESC order

	tests := []struct {
		name      string
		opType    string
		from      time.Time
		to        time.Time
		wantCount int
		wantErr   bool
	}{
		{
			name:      "get all entries",
			opType:    "",
			from:      time.Time{},
			to:        time.Time{},
			wantCount: 5,
			wantErr:   false,
		},
		{
			name:      "filter by ingest type",
			opType:    "ingest",
			from:      time.Time{},
			to:        time.Time{},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "filter by query type",
			opType:    "query",
			from:      time.Time{},
			to:        time.Time{},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "filter by delete type",
			opType:    "delete",
			from:      time.Time{},
			to:        time.Time{},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "filter by config type",
			opType:    "config",
			from:      time.Time{},
			to:        time.Time{},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "filter by date range - all",
			opType:    "",
			from:      earliest.Add(-1 * time.Second),
			to:        latest.Add(1 * time.Second),
			wantCount: 5,
			wantErr:   false,
		},
		{
			name:      "filter by date range - future",
			opType:    "",
			from:      latest.Add(1 * time.Hour),
			to:        latest.Add(2 * time.Hour),
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "filter by type and date",
			opType:    "ingest",
			from:      earliest.Add(-1 * time.Second),
			to:        latest.Add(1 * time.Second),
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "nonexistent type",
			opType:    "nonexistent",
			from:      time.Time{},
			to:        time.Time{},
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := store.GetAuditLog(ctx, tt.opType, tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAuditLog() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(entries) != tt.wantCount {
				t.Errorf("GetAuditLog() got %d entries, want %d", len(entries), tt.wantCount)
			}

			// Verify entries are sorted by timestamp DESC
			for i := 1; i < len(entries); i++ {
				if entries[i].Timestamp.After(entries[i-1].Timestamp) {
					t.Errorf("Entries not sorted by timestamp DESC: entry[%d] (%v) is after entry[%d] (%v)",
						i, entries[i].Timestamp, i-1, entries[i-1].Timestamp)
				}
			}

			// Verify all entries match the filter
			if tt.opType != "" {
				for _, entry := range entries {
					if entry.OperationType != tt.opType {
						t.Errorf("Entry has wrong operation type: got %s, want %s", entry.OperationType, tt.opType)
					}
				}
			}
		})
	}
}

func TestGetAuditLogFields(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test-audit-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Initialize store
	store, err := NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add a test entry with all fields populated
	opType := "ingest"
	details := "Ingested test-document.txt (2048 bytes)"
	userCtx := "test-user-session"

	err = store.AddAuditEntry(ctx, opType, details, userCtx)
	if err != nil {
		t.Fatalf("Failed to add audit entry: %v", err)
	}

	// Retrieve the entry
	entries, err := store.GetAuditLog(ctx, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("Failed to get audit log: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]

	// Verify all fields
	if entry.ID == 0 {
		t.Error("Entry ID should not be 0")
	}

	if entry.OperationType != opType {
		t.Errorf("OperationType = %s, want %s", entry.OperationType, opType)
	}

	if entry.Details != details {
		t.Errorf("Details = %s, want %s", entry.Details, details)
	}

	if entry.UserContext != userCtx {
		t.Errorf("UserContext = %s, want %s", entry.UserContext, userCtx)
	}

	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// Verify timestamp is recent (within last minute)
	if time.Since(entry.Timestamp) > time.Minute {
		t.Errorf("Timestamp is too old: %v", entry.Timestamp)
	}
}

func TestAuditLogConcurrency(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test-audit-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Initialize store
	store, err := NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Add entries concurrently
	numGoroutines := 10
	entriesPerGoroutine := 5

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < entriesPerGoroutine; j++ {
				err := store.AddAuditEntry(ctx, "test", "concurrent test", "goroutine")
				if err != nil {
					t.Errorf("Failed to add audit entry: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all entries were added
	entries, err := store.GetAuditLog(ctx, "test", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("Failed to get audit log: %v", err)
	}

	expectedCount := numGoroutines * entriesPerGoroutine
	if len(entries) != expectedCount {
		t.Errorf("Expected %d entries, got %d", expectedCount, len(entries))
	}
}
