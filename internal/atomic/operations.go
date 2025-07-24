// Package atomic provides atomic file operations to ensure data integrity
package atomic

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/AntoineArt/syncstation/internal/errors"
)

// FileWriter provides atomic file writing operations
type FileWriter struct {
	targetPath   string
	tempPath     string
	tempFile     *os.File
	committed    bool
	permissions  os.FileMode
}

// NewFileWriter creates a new atomic file writer
func NewFileWriter(targetPath string, permissions os.FileMode) (*FileWriter, error) {
	// Generate a unique temporary file name
	tempPath, err := generateTempPath(targetPath)
	if err != nil {
		return nil, errors.NewSyncError(
			"atomic_write",
			"",
			targetPath,
			fmt.Errorf("failed to generate temporary path: %w", err),
			errors.ErrCodeIOError,
		)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, errors.NewSyncError(
			"atomic_write",
			"",
			targetPath,
			fmt.Errorf("failed to create target directory: %w", err),
			errors.ErrCodeIOError,
		)
	}

	// Create temporary file
	tempFile, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions)
	if err != nil {
		return nil, errors.NewSyncError(
			"atomic_write",
			"",
			targetPath,
			fmt.Errorf("failed to create temporary file: %w", err),
			errors.ErrCodeIOError,
		)
	}

	return &FileWriter{
		targetPath:  targetPath,
		tempPath:    tempPath,
		tempFile:    tempFile,
		committed:   false,
		permissions: permissions,
	}, nil
}

// Write writes data to the temporary file
func (fw *FileWriter) Write(data []byte) (int, error) {
	if fw.tempFile == nil {
		return 0, errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("file writer is closed"),
			errors.ErrCodeIOError,
		)
	}

	n, err := fw.tempFile.Write(data)
	if err != nil {
		return n, errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("failed to write to temporary file: %w", err),
			errors.ErrCodeIOError,
		)
	}

	return n, nil
}

// WriteString writes a string to the temporary file
func (fw *FileWriter) WriteString(s string) (int, error) {
	return fw.Write([]byte(s))
}

// CopyFrom copies data from a reader to the temporary file
func (fw *FileWriter) CopyFrom(src io.Reader) (int64, error) {
	if fw.tempFile == nil {
		return 0, errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("file writer is closed"),
			errors.ErrCodeIOError,
		)
	}

	n, err := io.Copy(fw.tempFile, src)
	if err != nil {
		return n, errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("failed to copy to temporary file: %w", err),
			errors.ErrCodeIOError,
		)
	}

	return n, nil
}

// Sync flushes the temporary file to disk
func (fw *FileWriter) Sync() error {
	if fw.tempFile == nil {
		return errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("file writer is closed"),
			errors.ErrCodeIOError,
		)
	}

	if err := fw.tempFile.Sync(); err != nil {
		return errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("failed to sync temporary file: %w", err),
			errors.ErrCodeIOError,
		)
	}

	return nil
}

// Commit atomically moves the temporary file to the target location
func (fw *FileWriter) Commit() error {
	if fw.committed {
		return errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("file already committed"),
			errors.ErrCodeInternal,
		)
	}

	if fw.tempFile == nil {
		return errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("file writer is closed"),
			errors.ErrCodeIOError,
		)
	}

	// Sync and close the temporary file
	if err := fw.tempFile.Sync(); err != nil {
		fw.Rollback()
		return errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("failed to sync temporary file: %w", err),
			errors.ErrCodeIOError,
		)
	}

	if err := fw.tempFile.Close(); err != nil {
		fw.Rollback()
		return errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("failed to close temporary file: %w", err),
			errors.ErrCodeIOError,
		)
	}
	fw.tempFile = nil

	// Atomically move temporary file to target
	if err := os.Rename(fw.tempPath, fw.targetPath); err != nil {
		// Clean up temporary file
		os.Remove(fw.tempPath)
		return errors.NewSyncError(
			"atomic_write",
			"",
			fw.targetPath,
			fmt.Errorf("failed to move temporary file to target: %w", err),
			errors.ErrCodeIOError,
		)
	}

	fw.committed = true
	return nil
}

// Rollback removes the temporary file without committing
func (fw *FileWriter) Rollback() error {
	if fw.tempFile != nil {
		fw.tempFile.Close()
		fw.tempFile = nil
	}

	if fw.tempPath != "" {
		if err := os.Remove(fw.tempPath); err != nil && !os.IsNotExist(err) {
			return errors.NewSyncError(
				"atomic_write",
				"",
				fw.targetPath,
				fmt.Errorf("failed to remove temporary file: %w", err),
				errors.ErrCodeIOError,
			)
		}
	}

	return nil
}

// Close closes the file writer, rolling back if not committed
func (fw *FileWriter) Close() error {
	if !fw.committed {
		return fw.Rollback()
	}
	return nil
}

// AtomicOperation represents an atomic operation that can be committed or rolled back
type AtomicOperation interface {
	Execute() error
	Rollback() error
	IsExecuted() bool
}

// Transaction manages multiple atomic operations
type Transaction struct {
	operations []AtomicOperation
	executed   []AtomicOperation
	committed  bool
}

// NewTransaction creates a new transaction
func NewTransaction() *Transaction {
	return &Transaction{
		operations: make([]AtomicOperation, 0),
		executed:   make([]AtomicOperation, 0),
		committed:  false,
	}
}

// Add adds an operation to the transaction
func (tx *Transaction) Add(op AtomicOperation) {
	tx.operations = append(tx.operations, op)
}

// Execute executes all operations in the transaction
func (tx *Transaction) Execute() error {
	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}

	// Execute all operations
	for _, op := range tx.operations {
		if err := op.Execute(); err != nil {
			// Rollback all executed operations
			tx.rollbackExecuted()
			return fmt.Errorf("operation failed: %w", err)
		}
		tx.executed = append(tx.executed, op)
	}

	return nil
}

// Commit commits the transaction
func (tx *Transaction) Commit() error {
	if tx.committed {
		return fmt.Errorf("transaction already committed")
	}

	// Execute if not already executed
	if len(tx.executed) == 0 {
		if err := tx.Execute(); err != nil {
			return err
		}
	}

	tx.committed = true
	return nil
}

// Rollback rolls back all executed operations
func (tx *Transaction) Rollback() error {
	return tx.rollbackExecuted()
}

// rollbackExecuted rolls back all executed operations in reverse order
func (tx *Transaction) rollbackExecuted() error {
	var lastError error

	// Rollback in reverse order
	for i := len(tx.executed) - 1; i >= 0; i-- {
		if err := tx.executed[i].Rollback(); err != nil {
			lastError = err
		}
	}

	tx.executed = tx.executed[:0] // Clear executed operations
	return lastError
}

// FileOperation represents an atomic file operation
type FileOperation struct {
	operationType string
	srcPath      string
	dstPath      string
	backupPath   string
	executed     bool
	operation    func() error
	rollback     func() error
}

// NewCopyOperation creates a new atomic copy operation
func NewCopyOperation(srcPath, dstPath string) *FileOperation {
	backupPath := ""
	executed := false

	operation := func() error {
		// Check if destination exists and create backup
		if _, err := os.Stat(dstPath); err == nil {
			var err error
			backupPath, err = generateTempPath(dstPath)
			if err != nil {
				return err
			}
			if err := os.Rename(dstPath, backupPath); err != nil {
				return err
			}
		}

		// Copy file atomically
		writer, err := NewFileWriter(dstPath, 0644)
		if err != nil {
			return err
		}
		defer writer.Close()

		srcFile, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		if _, err := writer.CopyFrom(srcFile); err != nil {
			return err
		}

		if err := writer.Commit(); err != nil {
			return err
		}

		executed = true
		return nil
	}

	rollbackFunc := func() error {
		if !executed {
			return nil
		}

		// Remove destination file
		if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
			return err
		}

		// Restore backup if it exists
		if backupPath != "" {
			if err := os.Rename(backupPath, dstPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			backupPath = ""
		}

		executed = false
		return nil
	}

	return &FileOperation{
		operationType: "copy",
		srcPath:      srcPath,
		dstPath:      dstPath,
		executed:     false,
		operation:    operation,
		rollback:     rollbackFunc,
	}
}

// NewMoveOperation creates a new atomic move operation
func NewMoveOperation(srcPath, dstPath string) *FileOperation {
	backupPath := ""
	executed := false

	operation := func() error {
		// Check if destination exists and create backup
		if _, err := os.Stat(dstPath); err == nil {
			var err error
			backupPath, err = generateTempPath(dstPath)
			if err != nil {
				return err
			}
			if err := os.Rename(dstPath, backupPath); err != nil {
				return err
			}
		}

		// Move file atomically
		if err := os.Rename(srcPath, dstPath); err != nil {
			// Restore backup if move failed
			if backupPath != "" {
				os.Rename(backupPath, dstPath)
			}
			return err
		}

		executed = true
		return nil
	}

	rollbackFunc := func() error {
		if !executed {
			return nil
		}

		// Move file back to original location
		if err := os.Rename(dstPath, srcPath); err != nil && !os.IsNotExist(err) {
			return err
		}

		// Restore backup if it exists
		if backupPath != "" {
			if err := os.Rename(backupPath, dstPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			backupPath = ""
		}

		executed = false
		return nil
	}

	return &FileOperation{
		operationType: "move",
		srcPath:      srcPath,
		dstPath:      dstPath,
		executed:     false,
		operation:    operation,
		rollback:     rollbackFunc,
	}
}

// Execute executes the file operation
func (fo *FileOperation) Execute() error {
	return fo.operation()
}

// Rollback rolls back the file operation
func (fo *FileOperation) Rollback() error {
	return fo.rollback()
}

// IsExecuted returns true if the operation has been executed
func (fo *FileOperation) IsExecuted() bool {
	return fo.executed
}

// Utility functions

// generateTempPath generates a unique temporary file path
func generateTempPath(targetPath string) (string, error) {
	dir := filepath.Dir(targetPath)
	name := filepath.Base(targetPath)
	
	// Generate random suffix
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		// Fallback to timestamp if random fails
		timestamp := time.Now().UnixNano()
		return filepath.Join(dir, fmt.Sprintf(".%s.tmp.%d", name, timestamp)), nil
	}

	return filepath.Join(dir, fmt.Sprintf(".%s.tmp.%x", name, suffix)), nil
}

// WriteFileAtomic writes data to a file atomically
func WriteFileAtomic(targetPath string, data []byte, permissions os.FileMode) error {
	writer, err := NewFileWriter(targetPath, permissions)
	if err != nil {
		return err
	}
	defer writer.Close()

	if _, err := writer.Write(data); err != nil {
		return err
	}

	return writer.Commit()
}

// CopyFileAtomic copies a file atomically
func CopyFileAtomic(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return errors.NewSyncError(
			"atomic_copy",
			"",
			srcPath,
			fmt.Errorf("failed to open source file: %w", err),
			errors.ErrCodeFileNotFound,
		)
	}
	defer srcFile.Close()

	// Get source file permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return errors.NewSyncError(
			"atomic_copy",
			"",
			srcPath,
			fmt.Errorf("failed to get source file info: %w", err),
			errors.ErrCodeIOError,
		)
	}

	writer, err := NewFileWriter(dstPath, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer writer.Close()

	if _, err := writer.CopyFrom(srcFile); err != nil {
		return err
	}

	return writer.Commit()
}

// MoveFileAtomic moves a file atomically
func MoveFileAtomic(srcPath, dstPath string) error {
	// Try direct rename first (works if on same filesystem)
	if err := os.Rename(srcPath, dstPath); err == nil {
		return nil
	}

	// Fall back to copy + delete
	if err := CopyFileAtomic(srcPath, dstPath); err != nil {
		return err
	}

	if err := os.Remove(srcPath); err != nil {
		// If we can't remove source, try to clean up destination
		os.Remove(dstPath)
		return errors.NewSyncError(
			"atomic_move",
			"",
			srcPath,
			fmt.Errorf("failed to remove source file after copy: %w", err),
			errors.ErrCodeIOError,
		)
	}

	return nil
}