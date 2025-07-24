package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitOperationCallback represents a callback function for git operations
type GitOperationCallback func(localConfig *LocalConfig, filePath string, operation string) error

// LocalConfig represents the local CLI configuration stored on each computer
type LocalConfig struct {
	CloudSyncDir    string            `json:"cloudSyncDir"`    // Path to cloud sync directory
	CurrentComputer string            `json:"currentComputer"` // ID of current computer
	LastSyncTimes   map[string]string `json:"lastSyncTimes"`   // item name -> last sync timestamp
	GitMode         bool              `json:"gitMode"`         // Whether cloud directory is a git repository
	GitRepoRoot     string            `json:"gitRepoRoot"`     // Root of git repository (if gitMode is true)
}

// SyncItem represents a configuration item that can be synced (stored in cloud)
type SyncItem struct {
	Name            string            `json:"name"`
	Type            string            `json:"type"`            // "file" or "folder"
	Paths           map[string]string `json:"paths"`           // computerID -> path
	ExcludePatterns []string          `json:"excludePatterns"` // patterns to exclude during sync
}

// SyncItemsData represents the cloud-stored sync items configuration
type SyncItemsData struct {
	SyncItems []*SyncItem `json:"syncItems"`
}

// FileState represents the local state tracking for a file
type FileState struct {
	LocalHash   string `json:"localHash"`
	ModTime     string `json:"modTime"` // RFC3339 format
	Size        int64  `json:"size"`
	LastChecked string `json:"lastChecked"` // RFC3339 format
}

// FileStatesData represents local file state cache
type FileStatesData struct {
	States map[string]map[string]*FileState `json:"states"` // item name -> file path -> file state
}

// ComputerFileInfo represents file info from a specific computer
type ComputerFileInfo struct {
	Hash    string `json:"hash"`
	ModTime string `json:"modTime"` // RFC3339 format
}

// FileMetadata represents cloud-stored file metadata
type FileMetadata struct {
	Computers    map[string]*ComputerFileInfo `json:"computers"`    // computer ID -> file info
	CloudHash    string                       `json:"cloudHash"`    // hash of current cloud file
	CloudModTime string                       `json:"cloudModTime"` // RFC3339 format
	LastUpdated  string                       `json:"lastUpdated"`  // RFC3339 format
	UpdatedBy    string                       `json:"updatedBy"`    // computer ID that last updated
}

// FileMetadataData represents all cloud-stored file metadata
type FileMetadataData struct {
	Metadata map[string]map[string]*FileMetadata `json:"metadata"` // item name -> file path -> metadata
}

// FileStatus represents the status of a file during sync operations
type FileStatus struct {
	Path         string    `json:"path"`
	LocalExists  bool      `json:"localExists"`
	CloudExists  bool      `json:"cloudExists"`
	LocalModTime time.Time `json:"localModTime"`
	CloudModTime time.Time `json:"cloudModTime"`
	LocalHash    string    `json:"localHash"`
	CloudHash    string    `json:"cloudHash"`
	Status       string    `json:"status"` // "synced", "local_newer", "cloud_newer", "conflict", "missing"
}

// NewLocalConfig creates a new local configuration
func NewLocalConfig() *LocalConfig {
	return &LocalConfig{
		LastSyncTimes: make(map[string]string),
	}
}

// LoadLocalConfig loads local configuration from file
func LoadLocalConfig(filename string) (*LocalConfig, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return NewLocalConfig(), nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config LocalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Initialize map if nil
	if config.LastSyncTimes == nil {
		config.LastSyncTimes = make(map[string]string)
	}

	return &config, nil
}

// SaveLocalConfig saves local configuration to file
func (c *LocalConfig) SaveLocalConfig(filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// GetSyncItemsPath returns the path to sync items in cloud storage
func (c *LocalConfig) GetSyncItemsPath() string {
	return filepath.Join(c.CloudSyncDir, "sync-items.json")
}

// GetFileMetadataPath returns the path to file metadata in cloud storage
func (c *LocalConfig) GetFileMetadataPath() string {
	return filepath.Join(c.CloudSyncDir, "file-metadata.json")
}

// GetCloudConfigsPath returns the path to configs folder in cloud storage
func (c *LocalConfig) GetCloudConfigsPath() string {
	return filepath.Join(c.CloudSyncDir, "configs")
}

// NewSyncItemsData creates a new sync items data structure
func NewSyncItemsData() *SyncItemsData {
	return &SyncItemsData{
		SyncItems: make([]*SyncItem, 0),
	}
}

// LoadSyncItemsData loads sync items from cloud storage
func LoadSyncItemsData(filename string) (*SyncItemsData, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return NewSyncItemsData(), nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var syncData SyncItemsData
	if err := json.Unmarshal(data, &syncData); err != nil {
		return nil, err
	}

	// Initialize slice if nil
	if syncData.SyncItems == nil {
		syncData.SyncItems = make([]*SyncItem, 0)
	}

	return &syncData, nil
}

// SaveSyncItemsData saves sync items to cloud storage
func (s *SyncItemsData) SaveSyncItemsData(filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// AddSyncItem adds a new sync item
func (s *SyncItemsData) AddSyncItem(name, itemType string, paths map[string]string, excludePatterns []string) error {
	// Check for duplicate names
	for _, item := range s.SyncItems {
		if item.Name == name {
			return fmt.Errorf("sync item with name '%s' already exists", name)
		}
	}

	syncItem := &SyncItem{
		Name:            name,
		Type:            itemType,
		Paths:           paths,
		ExcludePatterns: excludePatterns,
	}

	s.SyncItems = append(s.SyncItems, syncItem)
	return nil
}

// FindSyncItem finds a sync item by name
func (s *SyncItemsData) FindSyncItem(name string) *SyncItem {
	for _, item := range s.SyncItems {
		if item.Name == name {
			return item
		}
	}
	return nil
}

// GetCurrentComputerPath returns the path for the current computer for a given sync item
func (item *SyncItem) GetCurrentComputerPath(computerID string) string {
	if path, exists := item.Paths[computerID]; exists {
		return ExpandPath(path)
	}
	return ""
}

// GetCloudPath returns the cloud storage path for a sync item
func (item *SyncItem) GetCloudPath(cloudConfigsPath string) string {
	// Replace spaces and special characters with safe alternatives
	safeName := strings.ReplaceAll(item.Name, " ", "-")
	safeName = strings.ReplaceAll(safeName, "/", "-")
	return filepath.Join(cloudConfigsPath, safeName)
}

// NewFileStatesData creates a new file states data structure
func NewFileStatesData() *FileStatesData {
	return &FileStatesData{
		States: make(map[string]map[string]*FileState),
	}
}

// LoadFileStatesData loads file states from local cache
func LoadFileStatesData(filename string) (*FileStatesData, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return NewFileStatesData(), nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var statesData FileStatesData
	if err := json.Unmarshal(data, &statesData); err != nil {
		return nil, err
	}

	// Initialize map if nil
	if statesData.States == nil {
		statesData.States = make(map[string]map[string]*FileState)
	}

	return &statesData, nil
}

// SaveFileStatesData saves file states to local cache
func (f *FileStatesData) SaveFileStatesData(filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// UpdateFileState updates the state for a specific file
func (f *FileStatesData) UpdateFileState(itemName, filePath string, hash string, modTime time.Time, size int64) {
	if f.States[itemName] == nil {
		f.States[itemName] = make(map[string]*FileState)
	}

	f.States[itemName][filePath] = &FileState{
		LocalHash:   hash,
		ModTime:     modTime.Format(time.RFC3339),
		Size:        size,
		LastChecked: time.Now().Format(time.RFC3339),
	}
}

// GetFileState retrieves the state for a specific file
func (f *FileStatesData) GetFileState(itemName, filePath string) *FileState {
	if itemStates, exists := f.States[itemName]; exists {
		return itemStates[filePath]
	}
	return nil
}

// NewFileMetadataData creates a new file metadata data structure
func NewFileMetadataData() *FileMetadataData {
	return &FileMetadataData{
		Metadata: make(map[string]map[string]*FileMetadata),
	}
}

// LoadFileMetadataData loads file metadata from cloud storage
func LoadFileMetadataData(filename string) (*FileMetadataData, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return NewFileMetadataData(), nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var metadataData FileMetadataData
	if err := json.Unmarshal(data, &metadataData); err != nil {
		return nil, err
	}

	// Initialize map if nil
	if metadataData.Metadata == nil {
		metadataData.Metadata = make(map[string]map[string]*FileMetadata)
	}

	return &metadataData, nil
}

// SaveFileMetadataData saves file metadata to cloud storage
func (f *FileMetadataData) SaveFileMetadataData(filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// UpdateFileMetadata updates metadata for a specific file
func (f *FileMetadataData) UpdateFileMetadata(itemName, filePath, computerID, hash string, modTime time.Time) {
	if f.Metadata[itemName] == nil {
		f.Metadata[itemName] = make(map[string]*FileMetadata)
	}

	if f.Metadata[itemName][filePath] == nil {
		f.Metadata[itemName][filePath] = &FileMetadata{
			Computers: make(map[string]*ComputerFileInfo),
		}
	}

	metadata := f.Metadata[itemName][filePath]
	metadata.Computers[computerID] = &ComputerFileInfo{
		Hash:    hash,
		ModTime: modTime.Format(time.RFC3339),
	}
	metadata.LastUpdated = time.Now().Format(time.RFC3339)
	metadata.UpdatedBy = computerID
}

// GetFileMetadata retrieves metadata for a specific file
func (f *FileMetadataData) GetFileMetadata(itemName, filePath string) *FileMetadata {
	if itemMetadata, exists := f.Metadata[itemName]; exists {
		return itemMetadata[filePath]
	}
	return nil
}

// CalculateFileHash calculates SHA256 hash of a file
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

// ExpandPath expands ~ and environment variables in file paths
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if we can't expand
		}
		return filepath.Join(homeDir, path[2:])
	}

	// Expand environment variables
	return os.ExpandEnv(path)
}

// PathExists checks if a file or directory exists
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetFileInfo gets file information (size, mod time)
func GetFileInfo(path string) (int64, time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}, err
	}
	return info.Size(), info.ModTime(), nil
}

// saveToGitNotes saves content to git notes
func saveToGitNotes(repoPath, notesRef, content string) error {
	cmd := exec.Command("git", "notes", "--ref", notesRef, "add", "-f", "-m", content, "HEAD")
	cmd.Dir = repoPath
	_, err := cmd.Output()
	if err != nil {
		// If HEAD doesn't exist (empty repo), create an initial commit
		if strings.Contains(err.Error(), "bad revision 'HEAD'") {
			// Create initial commit
			initCmd := exec.Command("git", "commit", "--allow-empty", "-m", "Initial commit for syncstation metadata")
			initCmd.Dir = repoPath
			if initErr := initCmd.Run(); initErr != nil {
				return fmt.Errorf("failed to create initial commit: %w", initErr)
			}
			// Retry adding notes
			cmd = exec.Command("git", "notes", "--ref", notesRef, "add", "-f", "-m", content, "HEAD")
			cmd.Dir = repoPath
			_, err = cmd.Output()
		}
	}
	return err
}

// loadFromGitNotes loads content from git notes
func loadFromGitNotes(repoPath, notesRef string) (string, error) {
	cmd := exec.Command("git", "notes", "--ref", notesRef, "show", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		// Notes don't exist yet, return empty content
		return "", nil
	}
	return string(output), nil
}

// SaveFileMetadataDataGitAware saves file metadata using git notes when in git mode
func (f *FileMetadataData) SaveFileMetadataDataGitAware(localConfig *LocalConfig, filename string) error {
	if localConfig.GitMode && localConfig.GitRepoRoot != "" {
		// Save to git notes
		data, err := json.MarshalIndent(f, "", "  ")
		if err != nil {
			return err
		}

		return saveToGitNotes(localConfig.GitRepoRoot, "syncstation/file-metadata", string(data))
	}

	// Fall back to regular file storage
	return f.SaveFileMetadataData(filename)
}

// LoadFileMetadataDataGitAware loads file metadata using git notes when in git mode
func LoadFileMetadataDataGitAware(localConfig *LocalConfig, filename string) (*FileMetadataData, error) {
	if localConfig.GitMode && localConfig.GitRepoRoot != "" {
		// Load from git notes
		content, err := loadFromGitNotes(localConfig.GitRepoRoot, "syncstation/file-metadata")
		if err != nil {
			return nil, err
		}

		if content == "" {
			// No notes exist yet, return new data
			return NewFileMetadataData(), nil
		}

		var metadataData FileMetadataData
		if err := json.Unmarshal([]byte(content), &metadataData); err != nil {
			return nil, err
		}

		// Initialize map if nil
		if metadataData.Metadata == nil {
			metadataData.Metadata = make(map[string]map[string]*FileMetadata)
		}

		return &metadataData, nil
	}

	// Fall back to regular file loading
	return LoadFileMetadataData(filename)
}
