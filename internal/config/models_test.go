package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewLocalConfig(t *testing.T) {
	config := NewLocalConfig()
	if config == nil {
		t.Fatal("NewLocalConfig returned nil")
	}
	if config.LastSyncTimes == nil {
		t.Fatal("LastSyncTimes map not initialized")
	}
}

func TestLocalConfigSaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test_config.json")

	// Create and save config
	originalConfig := NewLocalConfig()
	originalConfig.CloudSyncDir = "/test/cloud"
	originalConfig.CurrentComputer = "test-computer"
	originalConfig.GitMode = true
	originalConfig.GitRepoRoot = "/test/git"
	originalConfig.LastSyncTimes["test-item"] = "2023-01-01T00:00:00Z"

	err = originalConfig.SaveLocalConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := LoadLocalConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded config
	if loadedConfig.CloudSyncDir != originalConfig.CloudSyncDir {
		t.Errorf("CloudSyncDir mismatch: got %s, want %s", loadedConfig.CloudSyncDir, originalConfig.CloudSyncDir)
	}
	if loadedConfig.CurrentComputer != originalConfig.CurrentComputer {
		t.Errorf("CurrentComputer mismatch: got %s, want %s", loadedConfig.CurrentComputer, originalConfig.CurrentComputer)
	}
	if loadedConfig.GitMode != originalConfig.GitMode {
		t.Errorf("GitMode mismatch: got %v, want %v", loadedConfig.GitMode, originalConfig.GitMode)
	}
	if loadedConfig.LastSyncTimes["test-item"] != originalConfig.LastSyncTimes["test-item"] {
		t.Errorf("LastSyncTimes mismatch: got %s, want %s", 
			loadedConfig.LastSyncTimes["test-item"], originalConfig.LastSyncTimes["test-item"])
	}
}

func TestLoadLocalConfigNonExistent(t *testing.T) {
	config, err := LoadLocalConfig("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("Expected no error for non-existent file, got: %v", err)
	}
	if config == nil {
		t.Fatal("Expected new config, got nil")
	}
	if config.LastSyncTimes == nil {
		t.Fatal("LastSyncTimes map not initialized")
	}
}

func TestSyncItemsData(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	syncItemsPath := filepath.Join(tempDir, "sync-items.json")

	// Create sync items data
	syncData := NewSyncItemsData()
	if syncData.SyncItems == nil {
		t.Fatal("SyncItems slice not initialized")
	}

	// Add sync items
	paths1 := map[string]string{"computer1": "/path1", "computer2": "/path2"}
	excludes1 := []string{"*.log", "*.tmp"}
	err = syncData.AddSyncItem("item1", "file", paths1, excludes1)
	if err != nil {
		t.Fatalf("Failed to add sync item: %v", err)
	}

	paths2 := map[string]string{"computer1": "/path3"}
	err = syncData.AddSyncItem("item2", "folder", paths2, nil)
	if err != nil {
		t.Fatalf("Failed to add sync item: %v", err)
	}

	// Test duplicate name
	err = syncData.AddSyncItem("item1", "file", paths1, nil)
	if err == nil {
		t.Fatal("Expected error for duplicate name, got nil")
	}

	// Save and load
	err = syncData.SaveSyncItemsData(syncItemsPath)
	if err != nil {
		t.Fatalf("Failed to save sync items: %v", err)
	}

	loadedData, err := LoadSyncItemsData(syncItemsPath)
	if err != nil {
		t.Fatalf("Failed to load sync items: %v", err)
	}

	// Verify loaded data
	if len(loadedData.SyncItems) != 2 {
		t.Errorf("Expected 2 sync items, got %d", len(loadedData.SyncItems))
	}

	item1 := loadedData.FindSyncItem("item1")
	if item1 == nil {
		t.Fatal("item1 not found")
	}
	if item1.Type != "file" {
		t.Errorf("item1 type mismatch: got %s, want file", item1.Type)
	}
	if len(item1.ExcludePatterns) != 2 {
		t.Errorf("item1 exclude patterns count mismatch: got %d, want 2", len(item1.ExcludePatterns))
	}

	item2 := loadedData.FindSyncItem("item2")
	if item2 == nil {
		t.Fatal("item2 not found")
	}
	if item2.Type != "folder" {
		t.Errorf("item2 type mismatch: got %s, want folder", item2.Type)
	}

	nonExistent := loadedData.FindSyncItem("nonexistent")
	if nonExistent != nil {
		t.Error("Expected nil for non-existent item")
	}
}

func TestSyncItemGetCurrentComputerPath(t *testing.T) {
	item := &SyncItem{
		Name: "test-item",
		Type: "file",
		Paths: map[string]string{
			"computer1": "/path/to/file1",
			"computer2": "~/path/to/file2",
		},
	}

	path1 := item.GetCurrentComputerPath("computer1")
	if path1 != "/path/to/file1" {
		t.Errorf("Expected /path/to/file1, got %s", path1)
	}

	path2 := item.GetCurrentComputerPath("computer2")
	homeDir, _ := os.UserHomeDir()
	expectedPath2 := filepath.Join(homeDir, "path/to/file2")
	if path2 != expectedPath2 {
		t.Errorf("Expected %s, got %s", expectedPath2, path2)
	}

	path3 := item.GetCurrentComputerPath("nonexistent")
	if path3 != "" {
		t.Errorf("Expected empty string for non-existent computer, got %s", path3)
	}
}

func TestSyncItemGetCloudPath(t *testing.T) {
	item := &SyncItem{
		Name: "Test Item With Spaces",
		Type: "file",
	}

	cloudPath := item.GetCloudPath("/cloud/configs")
	expected := filepath.Join("/cloud/configs", "Test-Item-With-Spaces")
	if cloudPath != expected {
		t.Errorf("Expected %s, got %s", expected, cloudPath)
	}

	item.Name = "item/with/slashes"
	cloudPath = item.GetCloudPath("/cloud/configs")
	expected = filepath.Join("/cloud/configs", "item-with-slashes")
	if cloudPath != expected {
		t.Errorf("Expected %s, got %s", expected, cloudPath)
	}
}

func TestFileStatesData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	statesPath := filepath.Join(tempDir, "file-states.json")

	// Create file states data
	statesData := NewFileStatesData()
	if statesData.States == nil {
		t.Fatal("States map not initialized")
	}

	// Update file state
	testTime := time.Now()
	statesData.UpdateFileState("item1", "/path/to/file", "hash123", testTime, 1024)

	// Get file state
	state := statesData.GetFileState("item1", "/path/to/file")
	if state == nil {
		t.Fatal("File state not found")
	}
	if state.LocalHash != "hash123" {
		t.Errorf("Hash mismatch: got %s, want hash123", state.LocalHash)
	}
	if state.Size != 1024 {
		t.Errorf("Size mismatch: got %d, want 1024", state.Size)
	}

	// Test non-existent state
	state = statesData.GetFileState("nonexistent", "/path")
	if state != nil {
		t.Error("Expected nil for non-existent state")
	}

	// Save and load
	err = statesData.SaveFileStatesData(statesPath)
	if err != nil {
		t.Fatalf("Failed to save file states: %v", err)
	}

	loadedData, err := LoadFileStatesData(statesPath)
	if err != nil {
		t.Fatalf("Failed to load file states: %v", err)
	}

	// Verify loaded data
	loadedState := loadedData.GetFileState("item1", "/path/to/file")
	if loadedState == nil {
		t.Fatal("Loaded file state not found")
	}
	if loadedState.LocalHash != "hash123" {
		t.Errorf("Loaded hash mismatch: got %s, want hash123", loadedState.LocalHash)
	}
}

func TestFileMetadataData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	metadataPath := filepath.Join(tempDir, "file-metadata.json")

	// Create file metadata data
	metadataData := NewFileMetadataData()
	if metadataData.Metadata == nil {
		t.Fatal("Metadata map not initialized")
	}

	// Update file metadata
	testTime := time.Now()
	metadataData.UpdateFileMetadata("item1", "/path/to/file", "computer1", "hash123", testTime)

	// Get file metadata
	metadata := metadataData.GetFileMetadata("item1", "/path/to/file")
	if metadata == nil {
		t.Fatal("File metadata not found")
	}
	if metadata.UpdatedBy != "computer1" {
		t.Errorf("UpdatedBy mismatch: got %s, want computer1", metadata.UpdatedBy)
	}
	if metadata.Computers["computer1"].Hash != "hash123" {
		t.Errorf("Hash mismatch: got %s, want hash123", metadata.Computers["computer1"].Hash)
	}

	// Test non-existent metadata
	metadata = metadataData.GetFileMetadata("nonexistent", "/path")
	if metadata != nil {
		t.Error("Expected nil for non-existent metadata")
	}

	// Save and load
	err = metadataData.SaveFileMetadataData(metadataPath)
	if err != nil {
		t.Fatalf("Failed to save file metadata: %v", err)
	}

	loadedData, err := LoadFileMetadataData(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load file metadata: %v", err)
	}

	// Verify loaded data
	loadedMetadata := loadedData.GetFileMetadata("item1", "/path/to/file")
	if loadedMetadata == nil {
		t.Fatal("Loaded file metadata not found")
	}
	if loadedMetadata.UpdatedBy != "computer1" {
		t.Errorf("Loaded UpdatedBy mismatch: got %s, want computer1", loadedMetadata.UpdatedBy)
	}
}

func TestCalculateFileHash(t *testing.T) {
	// Create temporary file
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Calculate hash
	hash, err := CalculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}

	// Verify hash format
	if len(hash) == 0 {
		t.Fatal("Hash is empty")
	}
	if hash[:7] != "sha256:" {
		t.Errorf("Hash should start with 'sha256:', got: %s", hash[:7])
	}

	// Test with same content should give same hash
	hash2, err := CalculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate hash again: %v", err)
	}
	if hash != hash2 {
		t.Errorf("Hash mismatch for same file: %s != %s", hash, hash2)
	}

	// Test with non-existent file
	_, err = CalculateFileHash("/nonexistent/file")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	// Test tilde expansion
	expanded := ExpandPath("~/test/path")
	expected := filepath.Join(homeDir, "test/path")
	if expanded != expected {
		t.Errorf("Tilde expansion failed: got %s, want %s", expanded, expected)
	}

	// Test regular path
	regularPath := "/absolute/path"
	expanded = ExpandPath(regularPath)
	if expanded != regularPath {
		t.Errorf("Regular path changed: got %s, want %s", expanded, regularPath)
	}

	// Test environment variable expansion
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")
	
	expanded = ExpandPath("$TEST_VAR/path")
	expected = "test_value/path"
	if expanded != expected {
		t.Errorf("Env var expansion failed: got %s, want %s", expanded, expected)
	}
}

func TestPathExists(t *testing.T) {
	// Test with existing file (this test file)
	exists := PathExists("models_test.go")
	if !exists {
		// Try absolute path
		wd, _ := os.Getwd()
		testFile := filepath.Join(wd, "models_test.go")
		exists = PathExists(testFile)
	}
	
	// Test with non-existent file
	exists = PathExists("/definitely/nonexistent/path/file.txt")
	if exists {
		t.Error("PathExists returned true for non-existent path")
	}
}

func TestGetFileInfo(t *testing.T) {
	// Create temporary file
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Get file info
	size, modTime, err := GetFileInfo(testFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if size != int64(len(testContent)) {
		t.Errorf("Size mismatch: got %d, want %d", size, len(testContent))
	}

	if modTime.IsZero() {
		t.Error("ModTime is zero")
	}

	// Test with non-existent file
	_, _, err = GetFileInfo("/nonexistent/file")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// Benchmarks for performance testing
func BenchmarkCalculateFileHash(b *testing.B) {
	// Create temporary file
	tempDir, err := os.MkdirTemp("", "syncstation_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	// Create a larger file for meaningful benchmark
	content := make([]byte, 1024*1024) // 1MB
	for i := range content {
		content[i] = byte(i % 256)
	}
	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CalculateFileHash(testFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExpandPath(b *testing.B) {
	testPaths := []string{
		"~/test/path",
		"/absolute/path",
		"$HOME/relative/path",
		"regular/relative/path",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			ExpandPath(path)
		}
	}
}