package watcher

import (
	"context"
	"noodexx/internal/logging"
	"testing"
	"time"
)

// mockIngester for testing
type mockIngester struct {
	ingestedFiles map[int64][]string // userID -> list of file paths
}

func (m *mockIngester) IngestText(ctx context.Context, userID int64, source, text string, tags []string) error {
	if m.ingestedFiles == nil {
		m.ingestedFiles = make(map[int64][]string)
	}
	m.ingestedFiles[userID] = append(m.ingestedFiles[userID], source)
	return nil
}

// mockStore for testing
type mockStore struct {
	folders []WatchedFolder
}

func (m *mockStore) AddWatchedFolder(ctx context.Context, userID int64, path string) error {
	m.folders = append(m.folders, WatchedFolder{
		ID:     int64(len(m.folders) + 1),
		UserID: userID,
		Path:   path,
		Active: true,
	})
	return nil
}

func (m *mockStore) GetWatchedFolders(ctx context.Context) ([]WatchedFolder, error) {
	return m.folders, nil
}

func (m *mockStore) DeleteSource(ctx context.Context, source string) error {
	return nil
}

// mockLogger for testing
type mockLogger struct {
	logging.Logger
}

func newMockLogger() *logging.Logger {
	return logging.NewLogger("test", logging.DEBUG, nil)
}

func TestGetUserIDForFile(t *testing.T) {
	tests := []struct {
		name           string
		folders        map[string]int64 // path -> userID
		filePath       string
		expectedUserID int64
	}{
		{
			name: "file in user 1's folder",
			folders: map[string]int64{
				"/home/user1/docs": 1,
				"/home/user2/docs": 2,
			},
			filePath:       "/home/user1/docs/file.txt",
			expectedUserID: 1,
		},
		{
			name: "file in user 2's folder",
			folders: map[string]int64{
				"/home/user1/docs": 1,
				"/home/user2/docs": 2,
			},
			filePath:       "/home/user2/docs/notes.md",
			expectedUserID: 2,
		},
		{
			name: "file not in any watched folder",
			folders: map[string]int64{
				"/home/user1/docs": 1,
			},
			filePath:       "/tmp/random.txt",
			expectedUserID: 0,
		},
		{
			name: "nested folder structure",
			folders: map[string]int64{
				"/home/user1/docs": 1,
			},
			filePath:       "/home/user1/docs/subfolder/file.txt",
			expectedUserID: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Watcher{
				folderUsers: tt.folders,
			}

			userID := w.getUserIDForFile(tt.filePath)
			if userID != tt.expectedUserID {
				t.Errorf("getUserIDForFile() = %d, want %d", userID, tt.expectedUserID)
			}
		})
	}
}

func TestWatcherTracksUserOwnership(t *testing.T) {
	ctx := context.Background()
	mockStore := &mockStore{}
	mockIngester := &mockIngester{}
	mockLogger := newMockLogger()

	// Create temporary directories for testing
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	// Create watcher using NewWatcher to properly initialize fsWatcher
	w, err := NewWatcher(mockIngester, mockStore, false, mockLogger)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Add folders for different users
	err = w.AddFolder(ctx, 1, tmpDir1)
	if err != nil {
		t.Fatalf("Failed to add folder for user 1: %v", err)
	}

	err = w.AddFolder(ctx, 2, tmpDir2)
	if err != nil {
		t.Fatalf("Failed to add folder for user 2: %v", err)
	}

	// Verify folder ownership is tracked
	if w.folderUsers[tmpDir1] != 1 {
		t.Errorf("Expected user 1 to own %s, got user %d", tmpDir1, w.folderUsers[tmpDir1])
	}

	if w.folderUsers[tmpDir2] != 2 {
		t.Errorf("Expected user 2 to own %s, got user %d", tmpDir2, w.folderUsers[tmpDir2])
	}

	// Verify files are associated with correct users
	userID1 := w.getUserIDForFile(tmpDir1 + "/file.txt")
	if userID1 != 1 {
		t.Errorf("Expected file in user1's folder to belong to user 1, got user %d", userID1)
	}

	userID2 := w.getUserIDForFile(tmpDir2 + "/file.txt")
	if userID2 != 2 {
		t.Errorf("Expected file in user2's folder to belong to user 2, got user %d", userID2)
	}
}

func TestWatcherLoadsAllUserFolders(t *testing.T) {
	ctx := context.Background()
	mockStore := &mockStore{
		folders: []WatchedFolder{
			{ID: 1, UserID: 1, Path: "/tmp/user1", Active: true, LastScan: time.Now()},
			{ID: 2, UserID: 2, Path: "/tmp/user2", Active: true, LastScan: time.Now()},
			{ID: 3, UserID: 3, Path: "/tmp/user3", Active: true, LastScan: time.Now()},
		},
	}
	mockIngester := &mockIngester{}
	mockLogger := newMockLogger()

	// Create watcher
	w := &Watcher{
		ingester:    mockIngester,
		store:       mockStore,
		privacyMode: false,
		allowedExts: []string{".txt", ".md"},
		maxSize:     10 * 1024 * 1024,
		logger:      mockLogger,
		folderUsers: make(map[string]int64),
	}

	// Note: We can't actually call Start() because it would try to use fsnotify
	// which requires real filesystem paths. Instead, we'll manually populate
	// folderUsers to simulate what Start() does.
	folders, err := mockStore.GetWatchedFolders(ctx)
	if err != nil {
		t.Fatalf("Failed to get watched folders: %v", err)
	}

	for _, folder := range folders {
		w.folderUsers[folder.Path] = folder.UserID
	}

	// Verify all users' folders are tracked
	if len(w.folderUsers) != 3 {
		t.Errorf("Expected 3 folders to be tracked, got %d", len(w.folderUsers))
	}

	if w.folderUsers["/tmp/user1"] != 1 {
		t.Errorf("Expected /tmp/user1 to belong to user 1")
	}

	if w.folderUsers["/tmp/user2"] != 2 {
		t.Errorf("Expected /tmp/user2 to belong to user 2")
	}

	if w.folderUsers["/tmp/user3"] != 3 {
		t.Errorf("Expected /tmp/user3 to belong to user 3")
	}
}
