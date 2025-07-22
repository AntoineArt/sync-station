package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/AntoineArt/syncstation/internal/config"
	"github.com/AntoineArt/syncstation/internal/diff"
)

// SyncOperation represents a sync operation type
type SyncOperation int

const (
	SyncPush SyncOperation = iota // Local -> Cloud
	SyncPull                      // Cloud -> Local
	SyncSmart                     // Intelligent bidirectional sync
)

// SyncResult represents the result of a sync operation
type SyncResult struct {
	Operation     SyncOperation
	Success       bool
	FilesChanged  int
	FilesSkipped  int
	FilesErrored  int
	Errors        []string
	Message       string
}

// GitSafeOperationCallback represents a callback for git-safe operations
type GitSafeOperationCallback func(localConfig *config.LocalConfig, filePath string, operation func() error) error

// SyncEngine handles file synchronization operations
type SyncEngine struct {
	localConfig       *config.LocalConfig
	diffEngine        *diff.DiffEngine
	fileStatesPath    string
	cloudMetadataPath string
	gitCallback       config.GitOperationCallback // Callback for git operations
	gitSafeCallback   GitSafeOperationCallback    // Callback for git-safe operations
}

// NewSyncEngine creates a new sync engine
func NewSyncEngine(localConfig *config.LocalConfig, diffEngine *diff.DiffEngine) *SyncEngine {
	return &SyncEngine{
		localConfig:       localConfig,
		diffEngine:        diffEngine,
		fileStatesPath:    filepath.Join(getConfigDir(localConfig), "file-states.json"),
		cloudMetadataPath: localConfig.GetFileMetadataPath(),
		gitCallback:       nil, // Will be set by caller if needed
		gitSafeCallback:   nil, // Will be set by caller if needed
	}
}

// SetGitCallback sets the git operation callback
func (s *SyncEngine) SetGitCallback(callback config.GitOperationCallback) {
	s.gitCallback = callback
}

// SetGitSafeCallback sets the git-safe operation callback
func (s *SyncEngine) SetGitSafeCallback(callback GitSafeOperationCallback) {
	s.gitSafeCallback = callback
}

// getConfigDir returns the appropriate config directory for the platform
func getConfigDir(localConfig *config.LocalConfig) string {
	// This should match the logic in main.go getConfigDir()
	// For now, using a simple approach - could be enhanced
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		return filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "syncstation")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "syncstation")
}

// loadFileStates loads the local file states, creating empty data if file doesn't exist
func (s *SyncEngine) loadFileStates() (*config.FileStatesData, error) {
	if !config.PathExists(s.fileStatesPath) {
		return config.NewFileStatesData(), nil
	}
	return config.LoadFileStatesData(s.fileStatesPath)
}

// loadCloudMetadata loads the cloud metadata, creating empty data if file doesn't exist
func (s *SyncEngine) loadCloudMetadata() (*config.FileMetadataData, error) {
	// Use git-aware loading when in git mode
	return config.LoadFileMetadataDataGitAware(s.localConfig, s.cloudMetadataPath)
}

// updateFileMetadata updates both local and cloud metadata after a successful file operation
func (s *SyncEngine) updateFileMetadata(itemName, filePath string, fileInfo os.FileInfo, fileHash string) error {
	// Load current metadata
	fileStates, err := s.loadFileStates()
	if err != nil {
		return fmt.Errorf("failed to load file states: %w", err)
	}
	
	cloudMetadata, err := s.loadCloudMetadata()
	if err != nil {
		return fmt.Errorf("failed to load cloud metadata: %w", err)
	}
	
	// Update local file state
	fileStates.UpdateFileState(itemName, filePath, fileHash, fileInfo.ModTime(), fileInfo.Size())
	
	// Update cloud metadata
	cloudMetadata.UpdateFileMetadata(itemName, filePath, s.localConfig.CurrentComputer, fileHash, fileInfo.ModTime())
	
	// Save both files
	if err := fileStates.SaveFileStatesData(s.fileStatesPath); err != nil {
		return fmt.Errorf("failed to save file states: %w", err)
	}
	
	if err := cloudMetadata.SaveFileMetadataDataGitAware(s.localConfig, s.cloudMetadataPath); err != nil {
		return fmt.Errorf("failed to save cloud metadata: %w", err)
	}
	
	return nil
}

// updateCloudHash updates the cloud hash in metadata after a push operation
func (s *SyncEngine) updateCloudHash(itemName, filePath, cloudHash string, cloudModTime time.Time) error {
	cloudMetadata, err := s.loadCloudMetadata()
	if err != nil {
		return fmt.Errorf("failed to load cloud metadata: %w", err)
	}
	
	// Initialize maps if needed
	if cloudMetadata.Metadata == nil {
		cloudMetadata.Metadata = make(map[string]map[string]*config.FileMetadata)
	}
	if cloudMetadata.Metadata[itemName] == nil {
		cloudMetadata.Metadata[itemName] = make(map[string]*config.FileMetadata)
	}
	if cloudMetadata.Metadata[itemName][filePath] == nil {
		cloudMetadata.Metadata[itemName][filePath] = &config.FileMetadata{
			Computers: make(map[string]*config.ComputerFileInfo),
		}
	}
	
	// Update cloud-specific metadata
	cloudMetadata.Metadata[itemName][filePath].CloudHash = cloudHash
	cloudMetadata.Metadata[itemName][filePath].CloudModTime = cloudModTime.Format(time.RFC3339)
	cloudMetadata.Metadata[itemName][filePath].UpdatedBy = s.localConfig.CurrentComputer
	cloudMetadata.Metadata[itemName][filePath].LastUpdated = time.Now().Format(time.RFC3339)
	
	return cloudMetadata.SaveFileMetadataDataGitAware(s.localConfig, s.cloudMetadataPath)
}

// isFileChanged checks if a file has changed since last sync by comparing hashes
func (s *SyncEngine) isFileChanged(itemName, filePath string) (bool, error) {
	fileStates, err := s.loadFileStates()
	if err != nil {
		return true, err // Assume changed if we can't load states
	}
	
	existingState := fileStates.GetFileState(itemName, filePath)
	if existingState == nil {
		return true, nil // New file, consider it changed
	}
	
	currentHash, err := config.CalculateFileHash(filePath)
	if err != nil {
		return true, err // Assume changed if we can't calculate hash
	}
	
	return existingState.LocalHash != currentHash, nil
}

// SyncAll performs sync operation on all sync items
func (s *SyncEngine) SyncAll(operation SyncOperation, syncItems []*config.SyncItem) (*SyncResult, error) {
	result := &SyncResult{
		Operation: operation,
		Success:   true,
		Errors:    make([]string, 0),
	}

	for _, item := range syncItems {
		itemResult, err := s.SyncItem(operation, item)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", item.Name, err))
			result.FilesErrored++
			continue
		}

		result.FilesChanged += itemResult.FilesChanged
		result.FilesSkipped += itemResult.FilesSkipped
		result.FilesErrored += itemResult.FilesErrored
		result.Errors = append(result.Errors, itemResult.Errors...)
	}

	if result.FilesErrored > 0 {
		result.Success = false
	}

	result.Message = fmt.Sprintf("Sync complete: %d changed, %d skipped, %d errors", 
		result.FilesChanged, result.FilesSkipped, result.FilesErrored)
	
	return result, nil
}

// SyncItem performs sync operation on a single sync item  
func (s *SyncEngine) SyncItem(operation SyncOperation, item *config.SyncItem) (*SyncResult, error) {
	// Get local and cloud paths
	localPath := item.GetCurrentComputerPath(s.localConfig.CurrentComputer)
	if localPath == "" {
		return nil, fmt.Errorf("no path configured for computer '%s'", s.localConfig.CurrentComputer)
	}

	cloudPath := item.GetCloudPath(s.localConfig.GetCloudConfigsPath())

	// Perform sync based on operation type
	switch operation {
	case SyncPush:
		return s.pushItem(item, localPath, cloudPath)
	case SyncPull:
		return s.pullItem(item, localPath, cloudPath)
	case SyncSmart:
		return s.smartSyncItem(item, localPath, cloudPath)
	default:
		return nil, fmt.Errorf("unknown sync operation: %d", operation)
	}
}

// pushItem pushes a single item from local to cloud
func (s *SyncEngine) pushItem(item *config.SyncItem, localPath, cloudPath string) (*SyncResult, error) {
	result := &SyncResult{
		Operation: SyncPush,
		Success:   true,
		Errors:    make([]string, 0),
	}

	if !config.PathExists(localPath) {
		return nil, fmt.Errorf("local path does not exist: %s", localPath)
	}

	// For files, check if the file has actually changed to optimize sync
	if item.Type == "file" {
		changed, err := s.isFileChanged(item.Name, localPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to check file changes: %v", err))
		} else if !changed {
			result.Message = fmt.Sprintf("%s unchanged - skipped", item.Name)
			result.FilesSkipped = 1
			return result, nil
		}
	}


	// Check git staging before operation
	if s.gitCallback != nil {
		if err := s.gitCallback(s.localConfig, localPath, "pre_sync_backup"); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("git warning: %v", err))
		}
	}

	// Copy based on item type
	if item.Type == "file" {
		// Perform git-safe file operation
		copyOperation := func() error {
			return copyFile(localPath, cloudPath)
		}
		
		if s.gitSafeCallback != nil {
			if err := s.gitSafeCallback(s.localConfig, localPath, copyOperation); err != nil {
				return nil, fmt.Errorf("failed to copy file: %w", err)
			}
		} else {
			if err := copyOperation(); err != nil {
				return nil, fmt.Errorf("failed to copy file: %w", err)
			}
		}
		
		// Coordinate git staging for the synced file
		if s.gitCallback != nil {
			if err := s.gitCallback(s.localConfig, cloudPath, "sync_add"); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("git staging warning: %v", err))
			}
		}
		
		// Update metadata for the file
		localInfo, err := os.Stat(localPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to get file info: %v", err))
		} else {
			localHash, err := config.CalculateFileHash(localPath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to calculate hash: %v", err))
			} else {
				// Update local and computer metadata
				if err := s.updateFileMetadata(item.Name, localPath, localInfo, localHash); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to update metadata: %v", err))
				}
				
				// Update cloud metadata with cloud file hash
				cloudInfo, err := os.Stat(cloudPath)
				if err == nil {
					if err := s.updateCloudHash(item.Name, localPath, localHash, cloudInfo.ModTime()); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to update cloud hash: %v", err))
					}
				}
			}
		}
		
		result.FilesChanged = 1
	} else {
		if err := copyDir(localPath, cloudPath); err != nil {
			return nil, fmt.Errorf("failed to copy directory: %w", err)
		}
		
		// For directories, we'd need to recursively update metadata for all files
		// For now, just mark the directory operation as successful
		result.FilesChanged = 1
	}

	result.Message = fmt.Sprintf("Pushed %s to cloud", item.Name)
	return result, nil
}

// pullItem pulls a single item from cloud to local
func (s *SyncEngine) pullItem(item *config.SyncItem, localPath, cloudPath string) (*SyncResult, error) {
	result := &SyncResult{
		Operation: SyncPull,
		Success:   true,
		Errors:    make([]string, 0),
	}

	if !config.PathExists(cloudPath) {
		return nil, fmt.Errorf("cloud path does not exist: %s", cloudPath)
	}


	// Check git staging before operation
	if s.gitCallback != nil {
		if err := s.gitCallback(s.localConfig, localPath, "pre_sync_backup"); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("git warning: %v", err))
		}
	}

	// Copy based on item type
	if item.Type == "file" {
		// Perform git-safe file operation
		copyOperation := func() error {
			return copyFile(cloudPath, localPath)
		}
		
		if s.gitSafeCallback != nil {
			if err := s.gitSafeCallback(s.localConfig, localPath, copyOperation); err != nil {
				return nil, fmt.Errorf("failed to copy file: %w", err)
			}
		} else {
			if err := copyOperation(); err != nil {
				return nil, fmt.Errorf("failed to copy file: %w", err)
			}
		}
		
		// Coordinate git staging for the pulled file
		if s.gitCallback != nil {
			if err := s.gitCallback(s.localConfig, localPath, "sync_add"); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("git staging warning: %v", err))
			}
		}
		
		// Update metadata for the pulled file
		localInfo, err := os.Stat(localPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to get file info: %v", err))
		} else {
			localHash, err := config.CalculateFileHash(localPath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to calculate hash: %v", err))
			} else {
				if err := s.updateFileMetadata(item.Name, localPath, localInfo, localHash); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to update metadata: %v", err))
				}
			}
		}
		
		result.FilesChanged = 1
	} else {
		if err := copyDir(cloudPath, localPath); err != nil {
			return nil, fmt.Errorf("failed to copy directory: %w", err)
		}
		
		// For directories, mark as successful
		result.FilesChanged = 1
	}

	result.Message = fmt.Sprintf("Pulled %s from cloud", item.Name)
	return result, nil
}

// smartSyncItem performs intelligent bidirectional sync
func (s *SyncEngine) smartSyncItem(item *config.SyncItem, localPath, cloudPath string) (*SyncResult, error) {
	result := &SyncResult{
		Operation: SyncSmart,
		Success:   true,
		Errors:    make([]string, 0),
	}

	localExists := config.PathExists(localPath)
	cloudExists := config.PathExists(cloudPath)

	// Handle different scenarios
	if !localExists && !cloudExists {
		result.Message = fmt.Sprintf("Neither local nor cloud exists for %s", item.Name)
		return result, nil
	}

	if localExists && !cloudExists {
		// Local only - push to cloud
		return s.pushItem(item, localPath, cloudPath)
	}

	if !localExists && cloudExists {
		// Cloud only - pull to local
		return s.pullItem(item, localPath, cloudPath)
	}

	// Both exist - use intelligent hash-based comparison
	if item.Type == "file" {
		return s.smartSyncFile(item, localPath, cloudPath)
	} else {
		return s.smartSyncDirectory(item, localPath, cloudPath)
	}
}

// smartSyncFile performs intelligent sync for a single file using hash comparison
func (s *SyncEngine) smartSyncFile(item *config.SyncItem, localPath, cloudPath string) (*SyncResult, error) {
	result := &SyncResult{
		Operation: SyncSmart,
		Success:   true,
		Errors:    make([]string, 0),
	}

	// Calculate hashes for both files
	localHash, err := config.CalculateFileHash(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate local file hash: %w", err)
	}

	cloudHash, err := config.CalculateFileHash(cloudPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cloud file hash: %w", err)
	}

	// If hashes are the same, files are identical
	if localHash == cloudHash {
		// Update metadata if needed (in case we missed previous sync)
		localInfo, err := os.Stat(localPath)
		if err == nil {
			if err := s.updateFileMetadata(item.Name, localPath, localInfo, localHash); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to update metadata: %v", err))
			}
		}
		
		result.Message = fmt.Sprintf("%s is already in sync (hash match)", item.Name)
		result.FilesSkipped = 1
		return result, nil
	}

	// Files differ - check which one to sync based on timestamps and metadata
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat local file: %w", err)
	}

	cloudInfo, err := os.Stat(cloudPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat cloud file: %w", err)
	}

	// Load cloud metadata to check sync history
	cloudMetadata, err := s.loadCloudMetadata()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("warning: failed to load cloud metadata: %v", err))
	}

	// Check if we have metadata for this file
	var lastKnownCloudHash string
	if cloudMetadata != nil {
		if itemMetadata, exists := cloudMetadata.Metadata[item.Name]; exists {
			if fileMetadata, exists := itemMetadata[localPath]; exists {
				lastKnownCloudHash = fileMetadata.CloudHash
			}
		}
	}

	// Decision logic based on timestamps and metadata
	if lastKnownCloudHash != "" && lastKnownCloudHash == cloudHash {
		// Cloud hasn't changed since last sync, local must be newer
		result.Message = fmt.Sprintf("Local modified for %s - pushing to cloud", item.Name)
		return s.pushItem(item, localPath, cloudPath)
	} else if lastKnownCloudHash != "" && lastKnownCloudHash != cloudHash {
		// Cloud has changed since last sync
		if localInfo.ModTime().After(cloudInfo.ModTime()) {
			// Local is newer by timestamp - potential conflict
			result.Message = fmt.Sprintf("Conflict detected for %s - both files modified", item.Name)
			result.Errors = append(result.Errors, "Both local and cloud files have been modified since last sync")
			return result, nil
		} else {
			// Cloud is newer - pull from cloud
			result.Message = fmt.Sprintf("Cloud modified for %s - pulling to local", item.Name)
			return s.pullItem(item, localPath, cloudPath)
		}
	} else {
		// No previous metadata - fall back to timestamp comparison
		if localInfo.ModTime().After(cloudInfo.ModTime()) {
			result.Message = fmt.Sprintf("Local newer for %s - pushing to cloud", item.Name)
			return s.pushItem(item, localPath, cloudPath)
		} else if cloudInfo.ModTime().After(localInfo.ModTime()) {
			result.Message = fmt.Sprintf("Cloud newer for %s - pulling to local", item.Name)
			return s.pullItem(item, localPath, cloudPath)
		} else {
			// Same timestamp but different hashes - conflict
			result.Message = fmt.Sprintf("Conflict detected for %s - same timestamp, different content", item.Name)
			result.Errors = append(result.Errors, "Files have same timestamp but different content - manual resolution needed")
			return result, nil
		}
	}
}

// smartSyncDirectory performs intelligent sync for directories (simplified for now)
func (s *SyncEngine) smartSyncDirectory(item *config.SyncItem, localPath, cloudPath string) (*SyncResult, error) {
	result := &SyncResult{
		Operation: SyncSmart,
		Success:   true,
		Errors:    make([]string, 0),
	}

	// For directories, use timestamp comparison for now
	// TODO: Implement recursive file-by-file comparison
	localInfo, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat local directory: %w", err)
	}

	cloudInfo, err := os.Stat(cloudPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat cloud directory: %w", err)
	}

	if localInfo.ModTime().After(cloudInfo.ModTime()) {
		result.Message = fmt.Sprintf("Local directory newer for %s - pushing to cloud", item.Name)
		return s.pushItem(item, localPath, cloudPath)
	} else if cloudInfo.ModTime().After(localInfo.ModTime()) {
		result.Message = fmt.Sprintf("Cloud directory newer for %s - pulling to local", item.Name)
		return s.pullItem(item, localPath, cloudPath)
	} else {
		result.Message = fmt.Sprintf("Directory %s appears in sync", item.Name)
		result.FilesSkipped = 1
	}

	return result, nil
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir recursively copies a directory from src to dst
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

