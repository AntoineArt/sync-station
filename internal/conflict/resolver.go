// Package conflict provides conflict detection and resolution capabilities
package conflict

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AntoineArt/syncstation/internal/config"
	"github.com/AntoineArt/syncstation/internal/errors"
)

// ConflictType represents the type of conflict
type ConflictType int

const (
	ConflictTypeNone ConflictType = iota
	ConflictTypeBothModified
	ConflictTypeSameTime
	ConflictTypePermission
	ConflictTypeSize
)

func (ct ConflictType) String() string {
	switch ct {
	case ConflictTypeNone:
		return "none"
	case ConflictTypeBothModified:
		return "both_modified"
	case ConflictTypeSameTime:
		return "same_time"
	case ConflictTypePermission:
		return "permission"
	case ConflictTypeSize:
		return "size"
	default:
		return "unknown"
	}
}

// Conflict represents a sync conflict
type Conflict struct {
	ItemName       string                 `json:"itemName"`
	LocalPath      string                 `json:"localPath"`
	CloudPath      string                 `json:"cloudPath"`
	ConflictType   ConflictType           `json:"conflictType"`
	LocalInfo      *FileInfo              `json:"localInfo"`
	CloudInfo      *FileInfo              `json:"cloudInfo"`
	DetectedAt     time.Time              `json:"detectedAt"`
	Resolution     *ConflictResolution    `json:"resolution,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// FileInfo contains information about a file in conflict
type FileInfo struct {
	Hash    string      `json:"hash"`
	Size    int64       `json:"size"`
	ModTime time.Time   `json:"modTime"`
	Mode    os.FileMode `json:"mode"`
	Exists  bool        `json:"exists"`
}

// ConflictResolution represents how a conflict was resolved
type ConflictResolution struct {
	Strategy   ResolutionStrategy `json:"strategy"`
	ResolvedAt time.Time          `json:"resolvedAt"`
	ResolvedBy string             `json:"resolvedBy"`
	BackupPath string             `json:"backupPath,omitempty"`
	Notes      string             `json:"notes,omitempty"`
}

// ResolutionStrategy represents different ways to resolve conflicts
type ResolutionStrategy int

const (
	StrategyManual ResolutionStrategy = iota
	StrategyUseLocal
	StrategyUseCloud
	StrategyMerge
	StrategySkip
	StrategyBackupAndOverwrite
)

func (rs ResolutionStrategy) String() string {
	switch rs {
	case StrategyManual:
		return "manual"
	case StrategyUseLocal:
		return "use_local"
	case StrategyUseCloud:
		return "use_cloud"
	case StrategyMerge:
		return "merge"
	case StrategySkip:
		return "skip"
	case StrategyBackupAndOverwrite:
		return "backup_and_overwrite"
	default:
		return "unknown"
	}
}

// ConflictDetector detects conflicts between local and cloud files
type ConflictDetector struct {
	localConfig *config.LocalConfig
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector(localConfig *config.LocalConfig) *ConflictDetector {
	return &ConflictDetector{
		localConfig: localConfig,
	}
}

// DetectConflicts detects conflicts for a sync item
func (cd *ConflictDetector) DetectConflicts(item *config.SyncItem) (*Conflict, error) {
	localPath := item.GetCurrentComputerPath(cd.localConfig.CurrentComputer)
	if localPath == "" {
		return nil, errors.NewValidationError("local_path", "", "no path configured for current computer")
	}

	cloudPath := item.GetCloudPath(cd.localConfig.GetCloudConfigsPath())

	// Get file information
	localInfo := cd.getFileInfo(localPath)
	cloudInfo := cd.getFileInfo(cloudPath)

	// Determine conflict type
	conflictType := cd.determineConflictType(localInfo, cloudInfo, item)

	if conflictType == ConflictTypeNone {
		return nil, nil
	}

	return &Conflict{
		ItemName:     item.Name,
		LocalPath:    localPath,
		CloudPath:    cloudPath,
		ConflictType: conflictType,
		LocalInfo:    localInfo,
		CloudInfo:    cloudInfo,
		DetectedAt:   time.Now(),
		Metadata:     make(map[string]interface{}),
	}, nil
}

// getFileInfo retrieves file information
func (cd *ConflictDetector) getFileInfo(filePath string) *FileInfo {
	info := &FileInfo{
		Exists: config.PathExists(filePath),
	}

	if !info.Exists {
		return info
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		return info
	}

	info.Size = stat.Size()
	info.ModTime = stat.ModTime()
	info.Mode = stat.Mode()

	// Calculate hash if it's a regular file
	if stat.Mode().IsRegular() {
		hash, err := config.CalculateFileHash(filePath)
		if err == nil {
			info.Hash = hash
		}
	}

	return info
}

// determineConflictType determines the type of conflict
func (cd *ConflictDetector) determineConflictType(localInfo, cloudInfo *FileInfo, item *config.SyncItem) ConflictType {
	if !localInfo.Exists && !cloudInfo.Exists {
		return ConflictTypeNone
	}

	if !localInfo.Exists || !cloudInfo.Exists {
		return ConflictTypeNone // Not a conflict, just missing on one side
	}

	// Both files exist - check for conflicts
	if localInfo.Hash != "" && cloudInfo.Hash != "" {
		if localInfo.Hash == cloudInfo.Hash {
			return ConflictTypeNone // Files are identical
		}

		// Files are different - determine conflict type
		if localInfo.ModTime.Equal(cloudInfo.ModTime) {
			return ConflictTypeSameTime
		}

		// Check if both have been modified since last sync
		// This would require metadata to determine properly
		return ConflictTypeBothModified
	}

	// Size difference could indicate conflict
	if localInfo.Size != cloudInfo.Size {
		return ConflictTypeBothModified
	}

	return ConflictTypeNone
}

// InteractiveResolver provides interactive conflict resolution
type InteractiveResolver struct {
	localConfig *config.LocalConfig
	backupDir   string
	reader      *bufio.Reader
}

// NewInteractiveResolver creates a new interactive resolver
func NewInteractiveResolver(localConfig *config.LocalConfig, backupDir string) *InteractiveResolver {
	return &InteractiveResolver{
		localConfig: localConfig,
		backupDir:   backupDir,
		reader:      bufio.NewReader(os.Stdin),
	}
}

// ResolveConflict resolves a conflict interactively
func (ir *InteractiveResolver) ResolveConflict(conflict *Conflict) error {
	fmt.Printf("\n🔥 CONFLICT DETECTED: %s\n", conflict.ItemName)
	fmt.Printf("═══════════════════════════════════════════════════════\n")
	
	// Display conflict information
	ir.displayConflictInfo(conflict)
	
	// Get resolution strategy from user
	strategy, err := ir.getResolutionStrategy(conflict)
	if err != nil {
		return err
	}

	// Apply the resolution
	resolution, err := ir.applyResolution(conflict, strategy)
	if err != nil {
		return err
	}

	conflict.Resolution = resolution
	fmt.Printf("✅ Conflict resolved using strategy: %s\n", strategy.String())

	return nil
}

// displayConflictInfo displays detailed conflict information
func (ir *InteractiveResolver) displayConflictInfo(conflict *Conflict) {
	fmt.Printf("Conflict Type: %s\n", conflict.ConflictType.String())
	fmt.Printf("Local Path:    %s\n", conflict.LocalPath)
	fmt.Printf("Cloud Path:    %s\n", conflict.CloudPath)
	fmt.Printf("\n")

	// Local file info
	fmt.Printf("📁 LOCAL FILE:\n")
	if conflict.LocalInfo.Exists {
		fmt.Printf("   Size:     %d bytes\n", conflict.LocalInfo.Size)
		fmt.Printf("   Modified: %s\n", conflict.LocalInfo.ModTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Hash:     %s\n", conflict.LocalInfo.Hash[:16]+"...")
	} else {
		fmt.Printf("   File does not exist\n")
	}

	// Cloud file info
	fmt.Printf("\n☁️  CLOUD FILE:\n")
	if conflict.CloudInfo.Exists {
		fmt.Printf("   Size:     %d bytes\n", conflict.CloudInfo.Size)
		fmt.Printf("   Modified: %s\n", conflict.CloudInfo.ModTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Hash:     %s\n", conflict.CloudInfo.Hash[:16]+"...")
	} else {
		fmt.Printf("   File does not exist\n")
	}

	fmt.Printf("\n")
}

// getResolutionStrategy prompts user for resolution strategy
func (ir *InteractiveResolver) getResolutionStrategy(conflict *Conflict) (ResolutionStrategy, error) {
	fmt.Printf("Resolution Options:\n")
	fmt.Printf("1) Use Local File    - Keep the local version\n")
	fmt.Printf("2) Use Cloud File    - Keep the cloud version\n")
	fmt.Printf("3) View Differences  - Compare files side by side\n")
	fmt.Printf("4) Manual Edit       - Open files in editor for manual resolution\n")
	fmt.Printf("5) Backup & Use Local - Backup cloud file and use local\n")
	fmt.Printf("6) Skip              - Skip this file for now\n")
	fmt.Printf("\n")

	for {
		fmt.Printf("Choose resolution strategy (1-6): ")
		input, err := ir.reader.ReadString('\n')
		if err != nil {
			return StrategyManual, err
		}

		input = strings.TrimSpace(input)
		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("Please enter a valid number (1-6)\n")
			continue
		}

		switch choice {
		case 1:
			return StrategyUseLocal, nil
		case 2:
			return StrategyUseCloud, nil
		case 3:
			if err := ir.showDifferences(conflict); err != nil {
				fmt.Printf("Error showing differences: %v\n", err)
			}
			continue // Ask again after showing differences
		case 4:
			return StrategyManual, nil
		case 5:
			return StrategyBackupAndOverwrite, nil
		case 6:
			return StrategySkip, nil
		default:
			fmt.Printf("Please enter a number between 1 and 6\n")
			continue
		}
	}
}

// showDifferences shows differences between local and cloud files
func (ir *InteractiveResolver) showDifferences(conflict *Conflict) error {
	if !conflict.LocalInfo.Exists || !conflict.CloudInfo.Exists {
		fmt.Printf("Cannot show differences - one or both files don't exist\n")
		return nil
	}

	// Read file contents
	localContent, err := os.ReadFile(conflict.LocalPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	cloudContent, err := os.ReadFile(conflict.CloudPath)
	if err != nil {
		return fmt.Errorf("failed to read cloud file: %w", err)
	}

	// Simple side-by-side comparison
	fmt.Printf("\n📄 FILE COMPARISON:\n")
	fmt.Printf("═══════════════════════════════════════════════════════\n")
	fmt.Printf("LOCAL FILE (%s):\n", conflict.LocalPath)
	fmt.Printf("---\n%s\n---\n", string(localContent))
	fmt.Printf("\nCLOUD FILE (%s):\n", conflict.CloudPath)
	fmt.Printf("---\n%s\n---\n", string(cloudContent))
	fmt.Printf("═══════════════════════════════════════════════════════\n\n")

	return nil
}

// applyResolution applies the chosen resolution strategy
func (ir *InteractiveResolver) applyResolution(conflict *Conflict, strategy ResolutionStrategy) (*ConflictResolution, error) {
	resolution := &ConflictResolution{
		Strategy:   strategy,
		ResolvedAt: time.Now(),
		ResolvedBy: ir.localConfig.CurrentComputer,
	}

	switch strategy {
	case StrategyUseLocal:
		err := ir.copyFile(conflict.LocalPath, conflict.CloudPath)
		if err != nil {
			return nil, fmt.Errorf("failed to copy local file to cloud: %w", err)
		}
		resolution.Notes = "Used local file version"

	case StrategyUseCloud:
		err := ir.copyFile(conflict.CloudPath, conflict.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("failed to copy cloud file to local: %w", err)
		}
		resolution.Notes = "Used cloud file version"

	case StrategyBackupAndOverwrite:
		// Create backup of cloud file
		backupPath, err := ir.createBackup(conflict.CloudPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create backup: %w", err)
		}
		resolution.BackupPath = backupPath

		// Copy local file to cloud
		err = ir.copyFile(conflict.LocalPath, conflict.CloudPath)
		if err != nil {
			return nil, fmt.Errorf("failed to copy local file to cloud: %w", err)
		}
		resolution.Notes = fmt.Sprintf("Backed up cloud file to %s and used local version", backupPath)

	case StrategyManual:
		fmt.Printf("Please manually resolve the conflict and press Enter when done...")
		_, err := ir.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		resolution.Notes = "Manually resolved by user"

	case StrategySkip:
		resolution.Notes = "Skipped resolution"

	default:
		return nil, fmt.Errorf("unsupported resolution strategy: %v", strategy)
	}

	return resolution, nil
}

// copyFile copies a file from src to dst
func (ir *InteractiveResolver) copyFile(src, dst string) error {
	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Get source file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Write to destination with same permissions
	return os.WriteFile(dst, data, srcInfo.Mode())
}

// createBackup creates a backup copy of a file
func (ir *InteractiveResolver) createBackup(filePath string) (string, error) {
	if ir.backupDir == "" {
		ir.backupDir = filepath.Join(filepath.Dir(filePath), ".syncstation_backups")
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(ir.backupDir, 0755); err != nil {
		return "", err
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	baseName := filepath.Base(filePath)
	backupName := fmt.Sprintf("%s.%s.backup", baseName, timestamp)
	backupPath := filepath.Join(ir.backupDir, backupName)

	// Copy file to backup location
	err := ir.copyFile(filePath, backupPath)
	if err != nil {
		return "", err
	}

	return backupPath, nil
}

// AutoResolver provides automatic conflict resolution based on rules
type AutoResolver struct {
	localConfig *config.LocalConfig
	rules       []ResolutionRule
}

// ResolutionRule represents an automatic resolution rule
type ResolutionRule struct {
	Name        string
	Condition   func(*Conflict) bool
	Strategy    ResolutionStrategy
	Description string
}

// NewAutoResolver creates a new automatic resolver with default rules
func NewAutoResolver(localConfig *config.LocalConfig) *AutoResolver {
	resolver := &AutoResolver{
		localConfig: localConfig,
		rules:       make([]ResolutionRule, 0),
	}

	// Add default rules
	resolver.AddRule(ResolutionRule{
		Name: "use_newer_file",
		Condition: func(c *Conflict) bool {
			return c.LocalInfo.Exists && c.CloudInfo.Exists && 
				!c.LocalInfo.ModTime.Equal(c.CloudInfo.ModTime)
		},
		Strategy:    StrategyUseLocal, // This would be dynamic based on which is newer
		Description: "Use the file with the most recent modification time",
	})

	resolver.AddRule(ResolutionRule{
		Name: "use_larger_file",
		Condition: func(c *Conflict) bool {
			return c.LocalInfo.Exists && c.CloudInfo.Exists && 
				c.LocalInfo.Size != c.CloudInfo.Size
		},
		Strategy:    StrategyUseLocal, // This would be dynamic based on which is larger
		Description: "Use the larger file when sizes differ",
	})

	return resolver
}

// AddRule adds a resolution rule
func (ar *AutoResolver) AddRule(rule ResolutionRule) {
	ar.rules = append(ar.rules, rule)
}

// ResolveConflict attempts to resolve a conflict automatically
func (ar *AutoResolver) ResolveConflict(conflict *Conflict) (*ConflictResolution, error) {
	for _, rule := range ar.rules {
		if rule.Condition(conflict) {
			resolution := &ConflictResolution{
				Strategy:   rule.Strategy,
				ResolvedAt: time.Now(),
				ResolvedBy: "auto_resolver",
				Notes:      fmt.Sprintf("Automatically resolved using rule: %s", rule.Name),
			}

			// Apply the resolution (implementation would depend on strategy)
			// This is a simplified version
			return resolution, nil
		}
	}

	return nil, fmt.Errorf("no automatic resolution rule matched")
}