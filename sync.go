package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
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

// SyncEngine handles file synchronization operations
type SyncEngine struct {
	config     *Config
	diffEngine *DiffEngine
}

// NewSyncEngine creates a new sync engine
func NewSyncEngine(config *Config, diffEngine *DiffEngine) *SyncEngine {
	return &SyncEngine{
		config:     config,
		diffEngine: diffEngine,
	}
}

// SyncAll performs sync operation on all enabled sync items
func (s *SyncEngine) SyncAll(operation SyncOperation) (*SyncResult, error) {
	result := &SyncResult{
		Operation: operation,
		Success:   true,
		Errors:    make([]string, 0),
	}

	for _, syncItem := range s.config.SyncItems {
		if !syncItem.Enabled {
			continue
		}

		itemResult, err := s.SyncItem(syncItem, operation)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error syncing %s: %v", syncItem.Name, err))
			result.Success = false
			result.FilesErrored++
			continue
		}

		result.FilesChanged += itemResult.FilesChanged
		result.FilesSkipped += itemResult.FilesSkipped
		result.FilesErrored += itemResult.FilesErrored
		result.Errors = append(result.Errors, itemResult.Errors...)

		if !itemResult.Success {
			result.Success = false
		}
	}

	result.Message = fmt.Sprintf("Sync completed: %d changed, %d skipped, %d errors", 
		result.FilesChanged, result.FilesSkipped, result.FilesErrored)

	return result, nil
}

// SyncItem performs sync operation on a single sync item
func (s *SyncEngine) SyncItem(syncItem *SyncItem, operation SyncOperation) (*SyncResult, error) {
	result := &SyncResult{
		Operation: operation,
		Success:   true,
		Errors:    make([]string, 0),
	}

	localPath := s.config.GetCurrentComputerPath(syncItem)
	if localPath == "" {
		return nil, fmt.Errorf("no local path configured for current computer")
	}

	// Get file differences
	diffs, err := s.diffEngine.GetSyncItemDiff(syncItem)
	if err != nil {
		return nil, err
	}

	// Process each file based on operation type
	for fileName, diff := range diffs {
		switch operation {
		case SyncPush:
			err = s.pushFile(diff)
		case SyncPull:
			err = s.pullFile(diff)
		case SyncSmart:
			err = s.smartSync(diff)
		}

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error syncing %s: %v", fileName, err))
			result.FilesErrored++
			result.Success = false
		} else {
			if s.shouldSkipFile(diff, operation) {
				result.FilesSkipped++
			} else {
				result.FilesChanged++
			}
		}
	}

	result.Message = fmt.Sprintf("Synced %s: %d changed, %d skipped, %d errors", 
		syncItem.Name, result.FilesChanged, result.FilesSkipped, result.FilesErrored)

	return result, nil
}

// pushFile pushes a file from local to cloud
func (s *SyncEngine) pushFile(diff *FileDiff) error {
	if !diff.LocalExists {
		// File doesn't exist locally, remove from cloud if it exists
		if diff.CloudExists {
			return s.removeFile(diff.CloudPath)
		}
		return nil // Nothing to do
	}

	// Create cloud directory if it doesn't exist
	cloudDir := filepath.Dir(diff.CloudPath)
	if err := os.MkdirAll(cloudDir, 0755); err != nil {
		return fmt.Errorf("failed to create cloud directory: %v", err)
	}

	// Copy local file to cloud
	return s.copyFile(diff.LocalPath, diff.CloudPath)
}

// pullFile pulls a file from cloud to local
func (s *SyncEngine) pullFile(diff *FileDiff) error {
	if !diff.CloudExists {
		// File doesn't exist in cloud, remove from local if it exists
		if diff.LocalExists {
			return s.removeFile(diff.LocalPath)
		}
		return nil // Nothing to do
	}

	// Create local directory if it doesn't exist
	localDir := filepath.Dir(diff.LocalPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %v", err)
	}

	// Copy cloud file to local
	return s.copyFile(diff.CloudPath, diff.LocalPath)
}

// smartSync performs intelligent bidirectional sync
func (s *SyncEngine) smartSync(diff *FileDiff) error {
	switch diff.Status {
	case "same":
		// Files are identical, nothing to do
		return nil
	case "local_only":
		// File exists only locally, push to cloud
		return s.pushFile(diff)
	case "cloud_only":
		// File exists only in cloud, pull to local
		return s.pullFile(diff)
	case "local_newer":
		// Local file is newer, push to cloud
		return s.pushFile(diff)
	case "cloud_newer":
		// Cloud file is newer, pull to local
		return s.pullFile(diff)
	case "conflict":
		// Conflict detected, prefer local (can be made configurable)
		return s.pushFile(diff)
	default:
		return fmt.Errorf("unknown file status: %s", diff.Status)
	}
}

// shouldSkipFile determines if a file should be skipped based on its status
func (s *SyncEngine) shouldSkipFile(diff *FileDiff, operation SyncOperation) bool {
	switch operation {
	case SyncPush:
		return !diff.LocalExists
	case SyncPull:
		return !diff.CloudExists
	case SyncSmart:
		return diff.Status == "same"
	default:
		return false
	}
}

// copyFile copies a file from source to destination
func (s *SyncEngine) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// removeFile removes a file
func (s *SyncEngine) removeFile(path string) error {
	return os.Remove(path)
}

// CreateBackup creates a backup of files before syncing
func (s *SyncEngine) CreateBackup(syncItem *SyncItem) error {
	backupDir := filepath.Join(s.config.ConfigsPath, "backups", syncItem.Name, time.Now().Format("20060102_150405"))
	
	localPath := s.config.GetCurrentComputerPath(syncItem)
	if localPath == "" {
		return fmt.Errorf("no local path configured")
	}

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	// Copy all files to backup
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		backupPath := filepath.Join(backupDir, relPath)
		backupParentDir := filepath.Dir(backupPath)

		if err := os.MkdirAll(backupParentDir, 0755); err != nil {
			return err
		}

		return s.copyFile(path, backupPath)
	})
}

// GetSyncPreview returns a preview of what would be synced
func (s *SyncEngine) GetSyncPreview(syncItem *SyncItem, operation SyncOperation) (map[string]string, error) {
	preview := make(map[string]string)

	diffs, err := s.diffEngine.GetSyncItemDiff(syncItem)
	if err != nil {
		return nil, err
	}

	for fileName, diff := range diffs {
		switch operation {
		case SyncPush:
			if diff.LocalExists {
				preview[fileName] = "Push to cloud"
			} else if diff.CloudExists {
				preview[fileName] = "Remove from cloud"
			}
		case SyncPull:
			if diff.CloudExists {
				preview[fileName] = "Pull from cloud"
			} else if diff.LocalExists {
				preview[fileName] = "Remove from local"
			}
		case SyncSmart:
			switch diff.Status {
			case "same":
				preview[fileName] = "No changes needed"
			case "local_only":
				preview[fileName] = "Push to cloud"
			case "cloud_only":
				preview[fileName] = "Pull from cloud"
			case "local_newer":
				preview[fileName] = "Push newer local version"
			case "cloud_newer":
				preview[fileName] = "Pull newer cloud version"
			case "conflict":
				preview[fileName] = "Resolve conflict (prefer local)"
			}
		}
	}

	return preview, nil
}