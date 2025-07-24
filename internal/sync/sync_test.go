package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AntoineArt/syncstation/internal/config"
	"github.com/AntoineArt/syncstation/internal/diff"
)

func createTestConfig() *config.LocalConfig {
	return &config.LocalConfig{
		CloudSyncDir:    "/test/cloud",
		CurrentComputer: "test-computer",
		LastSyncTimes:   make(map[string]string),
		GitMode:         false,
		GitRepoRoot:     "",
	}
}

func createTestSyncItem(name, itemType string, localPath, cloudPath string) *config.SyncItem {
	return &config.SyncItem{
		Name: name,
		Type: itemType,
		Paths: map[string]string{
			"test-computer": localPath,
		},
		ExcludePatterns: []string{},
	}
}

func TestNewSyncEngine(t *testing.T) {
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	
	syncEngine := NewSyncEngine(localConfig, diffEngine)
	
	if syncEngine == nil {
		t.Fatal("NewSyncEngine returned nil")
	}
	if syncEngine.localConfig != localConfig {
		t.Error("localConfig not set correctly")
	}
	if syncEngine.diffEngine != diffEngine {
		t.Error("diffEngine not set correctly")
	}
}

func TestSyncEngineCallbacks(t *testing.T) {
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Test git callback
	gitCallback := func(localConfig *config.LocalConfig, filePath string, operation string) error {
		return nil
	}
	syncEngine.SetGitCallback(gitCallback)

	// Test git safe callback
	gitSafeCallback := func(localConfig *config.LocalConfig, filePath string, operation func() error) error {
		return operation()
	}
	syncEngine.SetGitSafeCallback(gitSafeCallback)

	// Verify callbacks are set
	if syncEngine.gitCallback == nil {
		t.Error("Git callback not set")
	}
	if syncEngine.gitSafeCallback == nil {
		t.Error("Git safe callback not set")
	}
}

func TestSyncResult(t *testing.T) {
	result := &SyncResult{
		Operation:    SyncPush,
		Success:      true,
		FilesChanged: 5,
		FilesSkipped: 2,
		FilesErrored: 1,
		Errors:       []string{"test error"},
		Message:      "test message",
	}

	if result.Operation != SyncPush {
		t.Error("Operation not set correctly")
	}
	if !result.Success {
		t.Error("Success not set correctly")
	}
	if result.FilesChanged != 5 {
		t.Error("FilesChanged not set correctly")
	}
	if len(result.Errors) != 1 {
		t.Error("Errors not set correctly")
	}
}

func TestCopyFile(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")
	err = os.MkdirAll(srcDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create source file
	srcFile := filepath.Join(srcDir, "test.txt")
	testContent := "Hello, World!"
	err = os.WriteFile(srcFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Copy file
	dstFile := filepath.Join(dstDir, "test.txt")
	err = copyFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify destination file exists and has correct content
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(dstContent) != testContent {
		t.Errorf("Content mismatch: got %s, want %s", string(dstContent), testContent)
	}

	// Verify permissions are copied
	srcInfo, err := os.Stat(srcFile)
	if err != nil {
		t.Fatal(err)
	}
	dstInfo, err := os.Stat(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("Permission mismatch: got %v, want %v", dstInfo.Mode(), srcInfo.Mode())
	}
}

func TestCopyDir(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")

	// Create source directory structure
	err = os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create files in source directory
	err = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Copy directory
	err = copyDir(srcDir, dstDir)
	if err != nil {
		t.Fatalf("Failed to copy directory: %v", err)
	}

	// Verify destination structure
	dstFile1 := filepath.Join(dstDir, "file1.txt")
	if !config.PathExists(dstFile1) {
		t.Error("file1.txt not copied")
	}

	dstFile2 := filepath.Join(dstDir, "subdir", "file2.txt")
	if !config.PathExists(dstFile2) {
		t.Error("subdir/file2.txt not copied")
	}

	// Verify content
	content1, err := os.ReadFile(dstFile1)
	if err != nil {
		t.Fatal(err)
	}
	if string(content1) != "content1" {
		t.Errorf("file1.txt content mismatch: got %s, want content1", string(content1))
	}

	content2, err := os.ReadFile(dstFile2)
	if err != nil {
		t.Fatal(err)
	}
	if string(content2) != "content2" {
		t.Errorf("file2.txt content mismatch: got %s, want content2", string(content2))
	}
}

func TestPushItemFile(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	localPath := filepath.Join(tempDir, "local", "config.txt")
	cloudPath := filepath.Join(tempDir, "cloud", "config.txt")

	// Create local file
	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		t.Fatal(err)
	}
	testContent := "test configuration"
	err = os.WriteFile(localPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync item
	item := createTestSyncItem("test-config", "file", localPath, cloudPath)

	// Push item
	result, err := syncEngine.pushItem(item, localPath, cloudPath)
	if err != nil {
		t.Fatalf("Failed to push item: %v", err)
	}

	if !result.Success {
		t.Error("Push should be successful")
	}
	if result.FilesChanged != 1 {
		t.Errorf("Expected 1 file changed, got %d", result.FilesChanged)
	}

	// Verify cloud file exists with correct content
	cloudContent, err := os.ReadFile(cloudPath)
	if err != nil {
		t.Fatalf("Failed to read cloud file: %v", err)
	}
	if string(cloudContent) != testContent {
		t.Errorf("Cloud file content mismatch: got %s, want %s", string(cloudContent), testContent)
	}
}

func TestPullItemFile(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	localPath := filepath.Join(tempDir, "local", "config.txt")
	cloudPath := filepath.Join(tempDir, "cloud", "config.txt")

	// Create cloud file
	err = os.MkdirAll(filepath.Dir(cloudPath), 0755)
	if err != nil {
		t.Fatal(err)
	}
	testContent := "cloud configuration"
	err = os.WriteFile(cloudPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync item
	item := createTestSyncItem("test-config", "file", localPath, cloudPath)

	// Pull item
	result, err := syncEngine.pullItem(item, localPath, cloudPath)
	if err != nil {
		t.Fatalf("Failed to pull item: %v", err)
	}

	if !result.Success {
		t.Error("Pull should be successful")
	}
	if result.FilesChanged != 1 {
		t.Errorf("Expected 1 file changed, got %d", result.FilesChanged)
	}

	// Verify local file exists with correct content
	localContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read local file: %v", err)
	}
	if string(localContent) != testContent {
		t.Errorf("Local file content mismatch: got %s, want %s", string(localContent), testContent)
	}
}

func TestSmartSyncFileIdentical(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	localPath := filepath.Join(tempDir, "local", "config.txt")
	cloudPath := filepath.Join(tempDir, "cloud", "config.txt")

	// Create identical files
	testContent := "identical content"
	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(filepath.Dir(cloudPath), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(localPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(cloudPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync item
	item := createTestSyncItem("test-config", "file", localPath, cloudPath)

	// Smart sync identical files
	result, err := syncEngine.smartSyncFile(item, localPath, cloudPath)
	if err != nil {
		t.Fatalf("Failed to smart sync identical files: %v", err)
	}

	if !result.Success {
		t.Error("Smart sync should be successful")
	}
	if result.FilesSkipped != 1 {
		t.Errorf("Expected 1 file skipped, got %d", result.FilesSkipped)
	}
	if result.FilesChanged != 0 {
		t.Errorf("Expected 0 files changed, got %d", result.FilesChanged)
	}
}

func TestSmartSyncFileDifferent(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	localPath := filepath.Join(tempDir, "local", "config.txt")
	cloudPath := filepath.Join(tempDir, "cloud", "config.txt")

	// Create different files with local newer
	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(filepath.Dir(cloudPath), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Cloud file first (older)
	err = os.WriteFile(cloudPath, []byte("cloud content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Local file after (newer)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	err = os.WriteFile(localPath, []byte("local content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync item
	item := createTestSyncItem("test-config", "file", localPath, cloudPath)

	// Smart sync different files
	result, err := syncEngine.smartSyncFile(item, localPath, cloudPath)
	if err != nil {
		t.Fatalf("Failed to smart sync different files: %v", err)
	}

	if !result.Success {
		t.Error("Smart sync should be successful")
	}
	if result.FilesChanged != 1 {
		t.Errorf("Expected 1 file changed, got %d", result.FilesChanged)
	}

	// Verify cloud file was updated with local content
	cloudContent, err := os.ReadFile(cloudPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(cloudContent) != "local content" {
		t.Errorf("Cloud file should be updated with local content, got: %s", string(cloudContent))
	}
}

func TestSmartSyncItemMissingLocal(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	localPath := filepath.Join(tempDir, "local", "config.txt")
	cloudPath := filepath.Join(tempDir, "cloud", "config.txt")

	// Create only cloud file
	err = os.MkdirAll(filepath.Dir(cloudPath), 0755)
	if err != nil {
		t.Fatal(err)
	}
	testContent := "cloud only content"
	err = os.WriteFile(cloudPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync item
	item := createTestSyncItem("test-config", "file", localPath, cloudPath)

	// Smart sync with missing local file
	result, err := syncEngine.smartSyncItem(item, localPath, cloudPath)
	if err != nil {
		t.Fatalf("Failed to smart sync with missing local: %v", err)
	}

	if !result.Success {
		t.Error("Smart sync should be successful")
	}

	// Verify local file was created with cloud content
	localContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Local file should be created: %v", err)
	}
	if string(localContent) != testContent {
		t.Errorf("Local file content mismatch: got %s, want %s", string(localContent), testContent)
	}
}

func TestSmartSyncItemMissingCloud(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	localPath := filepath.Join(tempDir, "local", "config.txt")
	cloudPath := filepath.Join(tempDir, "cloud", "config.txt")

	// Create only local file
	err = os.MkdirAll(filepath.Dir(localPath), 0755)
	if err != nil {
		t.Fatal(err)
	}
	testContent := "local only content"
	err = os.WriteFile(localPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync item
	item := createTestSyncItem("test-config", "file", localPath, cloudPath)

	// Smart sync with missing cloud file
	result, err := syncEngine.smartSyncItem(item, localPath, cloudPath)
	if err != nil {
		t.Fatalf("Failed to smart sync with missing cloud: %v", err)
	}

	if !result.Success {
		t.Error("Smart sync should be successful")
	}

	// Verify cloud file was created with local content
	cloudContent, err := os.ReadFile(cloudPath)
	if err != nil {
		t.Fatalf("Cloud file should be created: %v", err)
	}
	if string(cloudContent) != testContent {
		t.Errorf("Cloud file content mismatch: got %s, want %s", string(cloudContent), testContent)
	}
}

func TestSmartSyncItemBothMissing(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	localPath := filepath.Join(tempDir, "local", "config.txt")
	cloudPath := filepath.Join(tempDir, "cloud", "config.txt")

	// Don't create any files

	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync item
	item := createTestSyncItem("test-config", "file", localPath, cloudPath)

	// Smart sync with both files missing
	result, err := syncEngine.smartSyncItem(item, localPath, cloudPath)
	if err != nil {
		t.Fatalf("Failed to smart sync with both missing: %v", err)
	}

	if !result.Success {
		t.Error("Smart sync should be successful")
	}

	// No files should be created or changed
	if config.PathExists(localPath) {
		t.Error("Local file should not be created")
	}
	if config.PathExists(cloudPath) {
		t.Error("Cloud file should not be created")
	}
}

func TestSyncAll(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	localPath1 := filepath.Join(tempDir, "local", "config1.txt")
	cloudPath1 := filepath.Join(tempDir, "cloud", "config1.txt")
	localPath2 := filepath.Join(tempDir, "local", "config2.txt")
	cloudPath2 := filepath.Join(tempDir, "cloud", "config2.txt")

	err = os.MkdirAll(filepath.Dir(localPath1), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(localPath1, []byte("content1"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(localPath2, []byte("content2"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync items
	items := []*config.SyncItem{
		createTestSyncItem("config1", "file", localPath1, cloudPath1),
		createTestSyncItem("config2", "file", localPath2, cloudPath2),
	}

	// Sync all items
	result, err := syncEngine.SyncAll(SyncPush, items)
	if err != nil {
		t.Fatalf("Failed to sync all items: %v", err)
	}

	if !result.Success {
		t.Error("Sync all should be successful")
	}
	if result.FilesChanged != 2 {
		t.Errorf("Expected 2 files changed, got %d", result.FilesChanged)
	}

	// Verify both cloud files exist
	if !config.PathExists(cloudPath1) {
		t.Error("Cloud file 1 should exist")
	}
	if !config.PathExists(cloudPath2) {
		t.Error("Cloud file 2 should exist")
	}
}

func TestSyncItemWithInvalidPath(t *testing.T) {
	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync item with invalid computer path
	item := &config.SyncItem{
		Name: "test-config",
		Type: "file",
		Paths: map[string]string{
			"different-computer": "/some/path",
		},
	}

	// Try to sync item
	_, err := syncEngine.SyncItem(SyncPush, item)
	if err == nil {
		t.Error("Expected error for invalid computer path")
	}
}

// Benchmarks for performance testing
func BenchmarkCopyFile(b *testing.B) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcFile := filepath.Join(tempDir, "src.txt")
	// Create a reasonably sized file for benchmarking
	content := make([]byte, 1024*10) // 10KB
	for i := range content {
		content[i] = byte(i % 256)
	}
	err = os.WriteFile(srcFile, content, 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dstFile := filepath.Join(tempDir, fmt.Sprintf("dst_%d.txt", i))
		err := copyFile(srcFile, dstFile)
		if err != nil {
			b.Fatal(err)
		}
		// Clean up for next iteration
		os.Remove(dstFile)
	}
}

func BenchmarkSmartSyncFile(b *testing.B) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "syncstation_bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	localPath := filepath.Join(tempDir, "local.txt")
	cloudPath := filepath.Join(tempDir, "cloud.txt")

	// Create identical files for benchmark
	content := make([]byte, 1024*10) // 10KB
	for i := range content {
		content[i] = byte(i % 256)
	}
	err = os.WriteFile(localPath, content, 0644)
	if err != nil {
		b.Fatal(err)
	}
	err = os.WriteFile(cloudPath, content, 0644)
	if err != nil {
		b.Fatal(err)
	}

	// Create sync engine
	localConfig := createTestConfig()
	diffEngine := diff.NewDiffEngine()
	syncEngine := NewSyncEngine(localConfig, diffEngine)

	// Create sync item
	item := createTestSyncItem("bench-config", "file", localPath, cloudPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := syncEngine.smartSyncFile(item, localPath, cloudPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}