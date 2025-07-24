// Package backup provides backup and rollback functionality for syncstation
package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AntoineArt/syncstation/internal/config"
	"github.com/AntoineArt/syncstation/internal/errors"
)

// BackupEntry represents a single backup entry
type BackupEntry struct {
	ID          string    `json:"id"`
	ItemName    string    `json:"itemName"`
	OriginalPath string   `json:"originalPath"`
	BackupPath   string   `json:"backupPath"`
	Hash         string    `json:"hash"`
	Size         int64     `json:"size"`
	CreatedAt    time.Time `json:"createdAt"`
	CreatedBy    string    `json:"createdBy"`
	Reason       string    `json:"reason"`
	Tags         []string  `json:"tags"`
}

// BackupManifest represents the backup manifest file
type BackupManifest struct {
	Version   string         `json:"version"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Entries   []*BackupEntry `json:"entries"`
}

// BackupManager manages backup operations
type BackupManager struct {
	backupDir     string
	manifestPath  string
	maxBackups    int
	maxAge        time.Duration
	localConfig   *config.LocalConfig
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupDir string, localConfig *config.LocalConfig, options ...BackupOption) *BackupManager {
	bm := &BackupManager{
		backupDir:    backupDir,
		manifestPath: filepath.Join(backupDir, "manifest.json"),
		maxBackups:   50,  // Keep max 50 backups per item
		maxAge:       30 * 24 * time.Hour, // Keep backups for 30 days
		localConfig:  localConfig,
	}

	// Apply options
	for _, option := range options {
		option(bm)
	}

	// Ensure backup directory exists
	os.MkdirAll(backupDir, 0755)

	return bm
}

// BackupOption represents a configuration option for the backup manager
type BackupOption func(*BackupManager)

// WithMaxBackups sets the maximum number of backups to keep per item
func WithMaxBackups(max int) BackupOption {
	return func(bm *BackupManager) {
		bm.maxBackups = max
	}
}

// WithMaxAge sets the maximum age for backups
func WithMaxAge(age time.Duration) BackupOption {
	return func(bm *BackupManager) {
		bm.maxAge = age
	}
}

// BackupFile creates a backup of a file
func (bm *BackupManager) BackupFile(itemName, filePath, reason string, tags ...string) (*BackupEntry, error) {
	// Check if file exists
	if !config.PathExists(filePath) {
		return nil, errors.NewSyncError(
			"backup",
			itemName,
			filePath,
			fmt.Errorf("file does not exist"),
			errors.ErrCodeFileNotFound,
		)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, errors.NewSyncError(
			"backup",
			itemName,
			filePath,
			err,
			errors.ErrCodeIOError,
		)
	}

	// Calculate hash
	hash, err := config.CalculateFileHash(filePath)
	if err != nil {
		return nil, errors.NewSyncError(
			"backup",
			itemName,
			filePath,
			fmt.Errorf("failed to calculate hash: %w", err),
			errors.ErrCodeHashMismatch,
		)
	}

	// Generate backup ID and path
	backupID := bm.generateBackupID(itemName, hash)
	backupPath := filepath.Join(bm.backupDir, "files", backupID)

	// Check if backup already exists with same hash
	manifest, err := bm.loadManifest()
	if err != nil {
		return nil, err
	}

	for _, entry := range manifest.Entries {
		if entry.ItemName == itemName && entry.Hash == hash {
			// Backup with same content already exists, just update metadata
			entry.CreatedAt = time.Now()
			entry.Reason = reason
			entry.Tags = tags
			if err := bm.saveManifest(manifest); err != nil {
				return nil, err
			}
			return entry, nil
		}
	}

	// Create backup directory
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return nil, errors.NewSyncError(
			"backup",
			itemName,
			filePath,
			fmt.Errorf("failed to create backup directory: %w", err),
			errors.ErrCodeIOError,
		)
	}

	// Copy file to backup location
	if err := bm.copyFile(filePath, backupPath); err != nil {
		return nil, errors.NewSyncError(
			"backup",
			itemName,
			filePath,
			fmt.Errorf("failed to copy file to backup: %w", err),
			errors.ErrCodeIOError,
		)
	}

	// Create backup entry
	entry := &BackupEntry{
		ID:           backupID,
		ItemName:     itemName,
		OriginalPath: filePath,
		BackupPath:   backupPath,
		Hash:         hash,
		Size:         fileInfo.Size(),
		CreatedAt:    time.Now(),
		CreatedBy:    bm.localConfig.CurrentComputer,
		Reason:       reason,
		Tags:         tags,
	}

	// Add to manifest
	manifest.Entries = append(manifest.Entries, entry)
	manifest.UpdatedAt = time.Now()

	// Clean up old backups
	bm.cleanupOldBackups(manifest, itemName)

	// Save manifest
	if err := bm.saveManifest(manifest); err != nil {
		return nil, err
	}

	return entry, nil
}

// RestoreFile restores a file from backup
func (bm *BackupManager) RestoreFile(backupID, targetPath string) error {
	manifest, err := bm.loadManifest()
	if err != nil {
		return err
	}

	// Find backup entry
	var entry *BackupEntry
	for _, e := range manifest.Entries {
		if e.ID == backupID {
			entry = e
			break
		}
	}

	if entry == nil {
		return errors.NewSyncError(
			"restore",
			"",
			targetPath,
			fmt.Errorf("backup with ID %s not found", backupID),
			errors.ErrCodeFileNotFound,
		)
	}

	// Check if backup file exists
	if !config.PathExists(entry.BackupPath) {
		return errors.NewSyncError(
			"restore",
			entry.ItemName,
			targetPath,
			fmt.Errorf("backup file does not exist: %s", entry.BackupPath),
			errors.ErrCodeFileNotFound,
		)
	}

	// Create target directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return errors.NewSyncError(
			"restore",
			entry.ItemName,
			targetPath,
			fmt.Errorf("failed to create target directory: %w", err),
			errors.ErrCodeIOError,
		)
	}

	// Copy backup file to target
	if err := bm.copyFile(entry.BackupPath, targetPath); err != nil {
		return errors.NewSyncError(
			"restore",
			entry.ItemName,
			targetPath,
			fmt.Errorf("failed to restore file: %w", err),
			errors.ErrCodeIOError,
		)
	}

	return nil
}

// ListBackups lists all backups for an item
func (bm *BackupManager) ListBackups(itemName string) ([]*BackupEntry, error) {
	manifest, err := bm.loadManifest()
	if err != nil {
		return nil, err
	}

	var backups []*BackupEntry
	for _, entry := range manifest.Entries {
		if itemName == "" || entry.ItemName == itemName {
			backups = append(backups, entry)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// DeleteBackup deletes a specific backup
func (bm *BackupManager) DeleteBackup(backupID string) error {
	manifest, err := bm.loadManifest()
	if err != nil {
		return err
	}

	// Find and remove backup entry
	var entry *BackupEntry
	for i, e := range manifest.Entries {
		if e.ID == backupID {
			entry = e
			manifest.Entries = append(manifest.Entries[:i], manifest.Entries[i+1:]...)
			break
		}
	}

	if entry == nil {
		return errors.NewSyncError(
			"delete_backup",
			"",
			"",
			fmt.Errorf("backup with ID %s not found", backupID),
			errors.ErrCodeFileNotFound,
		)
	}

	// Delete backup file
	if config.PathExists(entry.BackupPath) {
		if err := os.Remove(entry.BackupPath); err != nil {
			return errors.NewSyncError(
				"delete_backup",
				entry.ItemName,
				entry.BackupPath,
				fmt.Errorf("failed to delete backup file: %w", err),
				errors.ErrCodeIOError,
			)
		}
	}

	// Save updated manifest
	manifest.UpdatedAt = time.Now()
	return bm.saveManifest(manifest)
}

// CleanupBackups removes old backups based on age and count limits
func (bm *BackupManager) CleanupBackups() error {
	manifest, err := bm.loadManifest()
	if err != nil {
		return err
	}

	// Group backups by item name
	itemBackups := make(map[string][]*BackupEntry)
	for _, entry := range manifest.Entries {
		itemBackups[entry.ItemName] = append(itemBackups[entry.ItemName], entry)
	}

	var toDelete []string
	now := time.Now()

	// Process each item's backups
	for itemName, backups := range itemBackups {
		// Sort by creation time (newest first)
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].CreatedAt.After(backups[j].CreatedAt)
		})

		// Mark old backups for deletion
		for i, backup := range backups {
			// Delete if too old
			if now.Sub(backup.CreatedAt) > bm.maxAge {
				toDelete = append(toDelete, backup.ID)
				continue
			}

			// Delete if exceeds max count (keep the newest ones)
			if i >= bm.maxBackups {
				toDelete = append(toDelete, backup.ID)
				continue
			}
		}
	}

	// Delete marked backups
	for _, backupID := range toDelete {
		if err := bm.DeleteBackup(backupID); err != nil {
			// Log error but continue cleanup
			fmt.Printf("Warning: failed to delete backup %s: %v\n", backupID, err)
		}
	}

	return nil
}

// GetBackupStats returns backup statistics
func (bm *BackupManager) GetBackupStats() (*BackupStats, error) {
	manifest, err := bm.loadManifest()
	if err != nil {
		return nil, err
	}

	stats := &BackupStats{
		TotalBackups: len(manifest.Entries),
		ItemCounts:   make(map[string]int),
		TotalSize:    0,
		OldestBackup: time.Now(),
		NewestBackup: time.Time{},
	}

	for _, entry := range manifest.Entries {
		stats.ItemCounts[entry.ItemName]++
		stats.TotalSize += entry.Size

		if entry.CreatedAt.Before(stats.OldestBackup) {
			stats.OldestBackup = entry.CreatedAt
		}
		if entry.CreatedAt.After(stats.NewestBackup) {
			stats.NewestBackup = entry.CreatedAt
		}
	}

	if len(manifest.Entries) == 0 {
		stats.OldestBackup = time.Time{}
	}

	return stats, nil
}

// BackupStats represents backup statistics
type BackupStats struct {
	TotalBackups int            `json:"totalBackups"`
	ItemCounts   map[string]int `json:"itemCounts"`
	TotalSize    int64          `json:"totalSize"`
	OldestBackup time.Time      `json:"oldestBackup"`
	NewestBackup time.Time      `json:"newestBackup"`
}

// loadManifest loads the backup manifest
func (bm *BackupManager) loadManifest() (*BackupManifest, error) {
	if !config.PathExists(bm.manifestPath) {
		// Create new manifest
		return &BackupManifest{
			Version:   "1.0",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Entries:   make([]*BackupEntry, 0),
		}, nil
	}

	data, err := os.ReadFile(bm.manifestPath)
	if err != nil {
		return nil, errors.NewConfigError(
			"backup_manifest",
			bm.manifestPath,
			err,
			errors.ErrCodeConfigLoad,
		)
	}

	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, errors.NewConfigError(
			"backup_manifest",
			bm.manifestPath,
			err,
			errors.ErrCodeConfigLoad,
		)
	}

	return &manifest, nil
}

// saveManifest saves the backup manifest
func (bm *BackupManager) saveManifest(manifest *BackupManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return errors.NewConfigError(
			"backup_manifest",
			bm.manifestPath,
			err,
			errors.ErrCodeConfigSave,
		)
	}

	return os.WriteFile(bm.manifestPath, data, 0644)
}

// generateBackupID generates a unique backup ID
func (bm *BackupManager) generateBackupID(itemName, hash string) string {
	timestamp := time.Now().Format("20060102_150405")
	safeName := strings.ReplaceAll(itemName, " ", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	hashPrefix := hash[7:15] // Use part of hash for uniqueness
	return fmt.Sprintf("%s_%s_%s", safeName, timestamp, hashPrefix)
}

// copyFile copies a file from src to dst
func (bm *BackupManager) copyFile(src, dst string) error {
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

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Copy file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// cleanupOldBackups removes old backups for an item
func (bm *BackupManager) cleanupOldBackups(manifest *BackupManifest, itemName string) {
	// Get backups for this item
	var itemBackups []*BackupEntry
	for _, entry := range manifest.Entries {
		if entry.ItemName == itemName {
			itemBackups = append(itemBackups, entry)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(itemBackups, func(i, j int) bool {
		return itemBackups[i].CreatedAt.After(itemBackups[j].CreatedAt)
	})

	// Mark entries for deletion
	var toDelete []string
	now := time.Now()

	for i, backup := range itemBackups {
		// Delete if too old
		if now.Sub(backup.CreatedAt) > bm.maxAge {
			toDelete = append(toDelete, backup.ID)
			continue
		}

		// Delete if exceeds max count
		if i >= bm.maxBackups {
			toDelete = append(toDelete, backup.ID)
			continue
		}
	}

	// Remove entries from manifest and delete files
	for _, backupID := range toDelete {
		for i, entry := range manifest.Entries {
			if entry.ID == backupID {
				// Delete backup file
				if config.PathExists(entry.BackupPath) {
					os.Remove(entry.BackupPath)
				}

				// Remove from manifest
				manifest.Entries = append(manifest.Entries[:i], manifest.Entries[i+1:]...)
				break
			}
		}
	}
}

// RollbackOperation represents a rollback operation
type RollbackOperation struct {
	ID          string             `json:"id"`
	ItemName    string             `json:"itemName"`
	BackupID    string             `json:"backupId"`
	TargetPath  string             `json:"targetPath"`
	ExecutedAt  time.Time          `json:"executedAt"`
	ExecutedBy  string             `json:"executedBy"`
	Status      RollbackStatus     `json:"status"`
	Error       string             `json:"error,omitempty"`
	PreRollback *BackupEntry       `json:"preRollback,omitempty"`
}

// RollbackStatus represents the status of a rollback operation
type RollbackStatus string

const (
	RollbackStatusPending   RollbackStatus = "pending"
	RollbackStatusSuccess   RollbackStatus = "success"
	RollbackStatusFailed    RollbackStatus = "failed"
	RollbackStatusRolledBack RollbackStatus = "rolled_back"
)

// ExecuteRollback executes a rollback operation with pre-rollback backup
func (bm *BackupManager) ExecuteRollback(itemName, backupID, targetPath string) (*RollbackOperation, error) {
	// Create pre-rollback backup of current state
	var preRollback *BackupEntry
	if config.PathExists(targetPath) {
		var err error
		preRollback, err = bm.BackupFile(itemName, targetPath, "pre_rollback", "rollback")
		if err != nil {
			return nil, fmt.Errorf("failed to create pre-rollback backup: %w", err)
		}
	}

	// Create rollback operation record
	operation := &RollbackOperation{
		ID:          bm.generateRollbackID(),
		ItemName:    itemName,
		BackupID:    backupID,
		TargetPath:  targetPath,
		ExecutedAt:  time.Now(),
		ExecutedBy:  bm.localConfig.CurrentComputer,
		Status:      RollbackStatusPending,
		PreRollback: preRollback,
	}

	// Execute the rollback
	err := bm.RestoreFile(backupID, targetPath)
	if err != nil {
		operation.Status = RollbackStatusFailed
		operation.Error = err.Error()
		return operation, err
	}

	operation.Status = RollbackStatusSuccess
	return operation, nil
}

// generateRollbackID generates a unique rollback operation ID
func (bm *BackupManager) generateRollbackID() string {
	return fmt.Sprintf("rollback_%d", time.Now().UnixNano())
}