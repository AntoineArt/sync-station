package syncstation

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AntoineArt/syncstation/internal/config"
	"github.com/AntoineArt/syncstation/internal/diff"
	"github.com/AntoineArt/syncstation/internal/sync"
	"github.com/AntoineArt/syncstation/internal/tui"
)

var (
	version = "1.0.3"

	// Global flags
	configDir string
	cloudDir  string
	computer  string
	gitMode   bool
	dryRun    bool
	verbose   bool
)

// Execute runs the root command
func Execute() {
	rootCmd := buildRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}

func buildRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "syncstation",
		Short: "A lightweight CLI/TUI for synchronizing configuration files",
		Long: `Syncstation is a cross-platform CLI/TUI application for synchronizing 
configuration files between multiple computers using your own cloud storage.

Features:
- Smart sync with hash-based change detection
- Git mode for version control
- Cross-platform support
- Interactive TUI interface`,
		Version: version,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "Custom config directory")
	rootCmd.PersistentFlags().StringVar(&computer, "computer", "", "Computer ID override")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")

	// Add commands
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(addCmd())
	rootCmd.AddCommand(syncCmd())
	rootCmd.AddCommand(pushCmd())
	rootCmd.AddCommand(pullCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(tuiCmd())
	rootCmd.AddCommand(removeCmd())
	rootCmd.AddCommand(configCmd())

	return rootCmd
}

func initCmd() *cobra.Command {
	var computerName string

	cmd := &cobra.Command{
		Use:   "init [cloud-dir]",
		Short: "Initialize Syncstation in a cloud directory",
		Long: `Initialize Syncstation configuration in your cloud sync directory.
If no directory is provided, the current directory will be used.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine cloud directory
			cloudSyncDir := "."
			if len(args) > 0 {
				cloudSyncDir = args[0]
			}

			// Convert to absolute path
			absCloudDir, err := filepath.Abs(cloudSyncDir)
			if err != nil {
				return fmt.Errorf("failed to resolve cloud directory path: %w", err)
			}

			// Check if directory exists
			if !config.PathExists(absCloudDir) {
				return fmt.Errorf("cloud directory does not exist: %s", absCloudDir)
			}

			// Get config directory
			configDirPath := getConfigDir()
			configPath := filepath.Join(configDirPath, "config.json")

			// Check if already initialized
			if config.PathExists(configPath) {
				localConfig, err := config.LoadLocalConfig(configPath)
				if err == nil && localConfig.CloudSyncDir != "" {
					fmt.Printf("Already initialized with cloud directory: %s\n", localConfig.CloudSyncDir)
					return nil
				}
			}

			// Generate computer ID
			defaultComputerID := getComputerID()
			computerID := defaultComputerID

			if computerName != "" {
				computerID = computerName
			} else if computer != "" {
				computerID = computer
			} else {
				// Prompt user for computer name
				fmt.Printf("💻 Computer name (default: %s): ", defaultComputerID)
				reader := bufio.NewReader(os.Stdin)
				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read input: %w", err)
				}

				input = strings.TrimSpace(input)
				if input != "" {
					computerID = input
				}
			}

			// Detect if cloud directory is a git repository
			isGitRepo := isGitRepository(absCloudDir)
			gitRepoRoot := ""
			if isGitRepo {
				gitRepoRoot = findGitRoot(absCloudDir)
			}

			// Create local config
			localConfig := config.NewLocalConfig()
			localConfig.CloudSyncDir = absCloudDir
			localConfig.CurrentComputer = computerID
			localConfig.GitMode = isGitRepo || gitMode
			localConfig.GitRepoRoot = gitRepoRoot

			// Save local config
			if err := localConfig.SaveLocalConfig(configPath); err != nil {
				return fmt.Errorf("failed to save local config: %w", err)
			}

			// Initialize cloud storage files (preserve existing data)
			syncItemsPath := localConfig.GetSyncItemsPath()
			syncItemsData, err := config.LoadSyncItemsData(syncItemsPath)
			if err != nil {
				return fmt.Errorf("failed to load or initialize sync items file: %w", err)
			}
			if err := syncItemsData.SaveSyncItemsData(syncItemsPath); err != nil {
				return fmt.Errorf("failed to save sync items file: %w", err)
			}

			// Initialize metadata (git-aware, preserve existing data)
			metadataData, err := config.LoadFileMetadataDataGitAware(localConfig, localConfig.GetFileMetadataPath())
			if err != nil {
				return fmt.Errorf("failed to load or initialize metadata file: %w", err)
			}
			if err := metadataData.SaveFileMetadataDataGitAware(localConfig, localConfig.GetFileMetadataPath()); err != nil {
				return fmt.Errorf("failed to save metadata file: %w", err)
			}

			fmt.Printf("✅ Initialized Syncstation in: %s\n", absCloudDir)
			fmt.Printf("📁 Computer ID: %s\n", computerID)
			if isGitRepo {
				fmt.Printf("🔄 Git mode enabled (detected git repository)\n")
			}
			fmt.Printf("\n💡 Next steps:\n")
			fmt.Printf("   syncstation add \"Config Name\" /path/to/config\n")
			fmt.Printf("   syncstation tui\n")

			return nil
		},
	}

	cmd.Flags().BoolVar(&gitMode, "git", false, "Force git mode even if not in a git repository")
	cmd.Flags().StringVar(&computerName, "name", "", "Set computer name (defaults to hostname)")
	return cmd
}

func addCmd() *cobra.Command {
	var excludePatterns []string

	cmd := &cobra.Command{
		Use:   "add <name> <path>",
		Short: "Add a sync item",
		Long: `Add a configuration file or directory to be synchronized.
The path should be the location on this computer.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			localPath := args[1]

			// Load configuration
			localConfig, err := loadConfig()
			if err != nil {
				return err
			}

			// Expand and convert to absolute path
			expandedPath := config.ExpandPath(localPath)
			if !config.PathExists(expandedPath) {
				return fmt.Errorf("path does not exist: %s", expandedPath)
			}

			// Convert to absolute path
			absolutePath, err := filepath.Abs(expandedPath)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}

			// Auto-detect type
			info, err := os.Stat(absolutePath)
			if err != nil {
				return fmt.Errorf("failed to stat path: %w", err)
			}
			itemType := "file"
			if info.IsDir() {
				itemType = "folder"
			}

			// Load sync items
			syncItems, err := config.LoadSyncItemsData(localConfig.GetSyncItemsPath())
			if err != nil {
				return fmt.Errorf("failed to load sync items: %w", err)
			}

			// Create paths map with current computer using absolute path
			paths := map[string]string{
				localConfig.CurrentComputer: absolutePath,
			}

			// Add sync item
			if err := syncItems.AddSyncItem(name, itemType, paths, excludePatterns); err != nil {
				return fmt.Errorf("failed to add sync item: %w", err)
			}

			// Save sync items
			if err := syncItems.SaveSyncItemsData(localConfig.GetSyncItemsPath()); err != nil {
				return fmt.Errorf("failed to save sync items: %w", err)
			}

			fmt.Printf("✅ Added sync item: %s\n", name)
			fmt.Printf("📁 Type: %s\n", itemType)
			fmt.Printf("📂 Path: %s\n", absolutePath)
			if len(excludePatterns) > 0 {
				fmt.Printf("🚫 Exclude patterns: %s\n", strings.Join(excludePatterns, ", "))
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&excludePatterns, "exclude", []string{}, "Patterns to exclude from sync")

	return cmd
}

func syncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [item-name]",
		Short: "Smart sync items",
		Long: `Perform intelligent bidirectional sync using hash comparison and timestamps.
If no item name is provided, all items will be synced.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return performSync(sync.SyncSmart, args)
		},
	}

	return cmd
}

func pushCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "push [item-name]",
		Short: "Push items from local to cloud",
		Long: `Push configuration files from local to cloud storage.
If no item name is provided, all items will be pushed.
Use --force to override conflict warnings.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return performSyncWithConflictCheck(sync.SyncPush, args, force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force push even when conflicts are detected")
	return cmd
}

func pullCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "pull [item-name]",
		Short: "Pull items from cloud to local",
		Long: `Pull configuration files from cloud storage to local.
If no item name is provided, all items will be pulled.
Use --force to override conflict warnings.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return performSyncWithConflictCheck(sync.SyncPull, args, force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force pull even when conflicts are detected")
	return cmd
}

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [item-name]",
		Short: "Show sync status",
		Long: `Display the synchronization status of items.
Shows whether files are in sync, have conflicts, or need updates.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			localConfig, err := loadConfig()
			if err != nil {
				return err
			}

			// Load sync items
			syncItems, err := config.LoadSyncItemsData(localConfig.GetSyncItemsPath())
			if err != nil {
				return fmt.Errorf("failed to load sync items: %w", err)
			}

			if len(syncItems.SyncItems) == 0 {
				fmt.Println("📭 No sync items configured")
				return nil
			}

			// Filter items if specific item requested
			itemsToCheck := syncItems.SyncItems
			if len(args) > 0 {
				itemName := args[0]
				item := syncItems.FindSyncItem(itemName)
				if item == nil {
					return fmt.Errorf("sync item not found: %s", itemName)
				}
				itemsToCheck = []*config.SyncItem{item}
			}

			// Display status header
			fmt.Printf("🔄 Sync Status - Computer: %s\n", localConfig.CurrentComputer)
			fmt.Printf("☁️  Cloud Directory: %s\n\n", localConfig.CloudSyncDir)

			// Create diff engine for status checking
			diffEngine := diff.NewDiffEngine()

			// Check each item
			for _, item := range itemsToCheck {
				localPath := item.GetCurrentComputerPath(localConfig.CurrentComputer)
				cloudPath := item.GetCloudPath(localConfig.GetCloudConfigsPath())

				fmt.Printf("📦 %s (%s)\n", item.Name, item.Type)
				fmt.Printf("   Local:  %s\n", getPathStatus(localPath))
				fmt.Printf("   Cloud:  %s\n", getPathStatus(cloudPath))

				// Get detailed status if both exist
				if config.PathExists(localPath) && config.PathExists(cloudPath) {
					if item.Type == "file" {
						fileDiff, err := diffEngine.CompareFiles(localPath, cloudPath)
						if err != nil {
							fmt.Printf("   Status: ❌ Error checking: %v\n", err)
						} else {
							fmt.Printf("   Status: %s\n", getStatusIcon(fileDiff.Status))
						}
					} else {
						fmt.Printf("   Status: 📁 Directory (detailed comparison not implemented)\n")
					}
				} else if !config.PathExists(localPath) && !config.PathExists(cloudPath) {
					fmt.Printf("   Status: ⚠️  Neither exists\n")
				} else if !config.PathExists(localPath) {
					fmt.Printf("   Status: ⬇️  Need to pull from cloud\n")
				} else {
					fmt.Printf("   Status: ⬆️  Need to push to cloud\n")
				}

				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all sync items",
		Long:  `Display all configured sync items with their paths and status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			localConfig, err := loadConfig()
			if err != nil {
				return err
			}

			// Load sync items
			syncItems, err := config.LoadSyncItemsData(localConfig.GetSyncItemsPath())
			if err != nil {
				return fmt.Errorf("failed to load sync items: %w", err)
			}

			if len(syncItems.SyncItems) == 0 {
				fmt.Println("📭 No sync items configured")
				fmt.Println("💡 Add items with: syncstation add \"Name\" /path/to/config")
				return nil
			}

			fmt.Printf("📦 Sync Items (%d total)\n\n", len(syncItems.SyncItems))

			for _, item := range syncItems.SyncItems {
				typeIcon := "📄"
				if item.Type == "folder" {
					typeIcon = "📁"
				}

				fmt.Printf("%s %s\n", typeIcon, item.Name)

				// Show paths for all computers
				if len(item.Paths) > 0 {
					for computerID, path := range item.Paths {
						marker := "  "
						if computerID == localConfig.CurrentComputer {
							marker = "▶ "
						}
						fmt.Printf("%s%s: %s\n", marker, computerID, path)
					}
				}

				// Show exclude patterns if any
				if len(item.ExcludePatterns) > 0 {
					fmt.Printf("   🚫 Excludes: %s\n", strings.Join(item.ExcludePatterns, ", "))
				}

				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

func tuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		Long:  `Launch the beautiful terminal user interface for interactive sync management.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.LaunchTUI()
		},
	}

	return cmd
}

func removeCmd() *cobra.Command {
	var global bool
	var deleteCloud bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a sync item",
		Long: `Remove a sync item from synchronization.

By default, only disables sync on this computer (other computers keep the item).
Use --global to remove from all computers' configurations.
Use --delete-cloud to also delete the cloud backup files.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Validate flags
			if global && deleteCloud {
				return fmt.Errorf("cannot use --global and --delete-cloud together")
			}

			// Load configuration
			localConfig, err := loadConfig()
			if err != nil {
				return err
			}

			// Load sync items
			syncItems, err := config.LoadSyncItemsData(localConfig.GetSyncItemsPath())
			if err != nil {
				return fmt.Errorf("failed to load sync items: %w", err)
			}

			// Find the item
			var targetItem *config.SyncItem
			var itemIndex int
			for i, item := range syncItems.SyncItems {
				if item.Name == name {
					targetItem = item
					itemIndex = i
					break
				}
			}

			if targetItem == nil {
				return fmt.Errorf("sync item not found: %s", name)
			}

			if deleteCloud {
				// Complete deletion: remove item + delete cloud files
				cloudPath := targetItem.GetCloudPath(localConfig.GetCloudConfigsPath())
				if config.PathExists(cloudPath) {
					if err := deleteCloudFiles(cloudPath, targetItem.Type); err != nil {
						fmt.Printf("⚠️  Warning: failed to delete cloud files at %s: %v\n", cloudPath, err)
					} else {
						fmt.Printf("🗑️  Deleted cloud backup files at: %s\n", cloudPath)
					}
				}

				// Clean up metadata for this item
				if err := cleanupItemMetadata(localConfig, targetItem); err != nil {
					fmt.Printf("⚠️  Warning: failed to cleanup metadata: %v\n", err)
				}

				// Remove item from configuration
				syncItems.SyncItems = append(syncItems.SyncItems[:itemIndex], syncItems.SyncItems[itemIndex+1:]...)

				// Save updated sync items
				if err := syncItems.SaveSyncItemsData(localConfig.GetSyncItemsPath()); err != nil {
					return fmt.Errorf("failed to save sync items: %w", err)
				}

				fmt.Printf("✅ Completely removed '%s' and deleted cloud backup files\n", name)
				return nil
			}

			if global {
				// Global removal: remove item from all computers but preserve cloud files
				// Clean up metadata for this item
				if err := cleanupItemMetadata(localConfig, targetItem); err != nil {
					fmt.Printf("⚠️  Warning: failed to cleanup metadata: %v\n", err)
				}

				syncItems.SyncItems = append(syncItems.SyncItems[:itemIndex], syncItems.SyncItems[itemIndex+1:]...)

				// Save updated sync items
				if err := syncItems.SaveSyncItemsData(localConfig.GetSyncItemsPath()); err != nil {
					return fmt.Errorf("failed to save sync items: %w", err)
				}

				fmt.Printf("✅ Removed sync item '%s' from all computers (cloud backup files preserved)\n", name)
				return nil
			}

			// Default: Local-only removal
			if len(targetItem.Paths) <= 1 {
				fmt.Printf("⚠️  Warning: '%s' only has one computer configured.\n", name)
				fmt.Printf("💡 Use 'syncstation remove \"%s\" --global' to remove completely.\n", name)
				return nil
			}

			if _, exists := targetItem.Paths[localConfig.CurrentComputer]; !exists {
				fmt.Printf("ℹ️  '%s' is not configured for this computer (%s)\n", name, localConfig.CurrentComputer)
				return nil
			}

			// Remove only this computer's path
			delete(targetItem.Paths, localConfig.CurrentComputer)

			// Save updated sync items
			if err := syncItems.SaveSyncItemsData(localConfig.GetSyncItemsPath()); err != nil {
				return fmt.Errorf("failed to save sync items: %w", err)
			}

			fmt.Printf("✅ Disabled sync for '%s' on this computer (%s)\n", name, localConfig.CurrentComputer)
			fmt.Printf("💡 Item remains active on other computers: %v\n", getComputerList(targetItem.Paths))
			return nil
		},
	}

	cmd.Flags().BoolVar(&global, "global", false, "Remove from all computers (instead of just this computer)")
	cmd.Flags().BoolVar(&deleteCloud, "delete-cloud", false, "Completely remove and delete cloud backup files (WARNING: permanent deletion)")
	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show configuration info",
		Long:  `Display current Syncstation configuration and paths.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := filepath.Join(getConfigDir(), "config.json")

			if !config.PathExists(configPath) {
				fmt.Println("❌ Syncstation not initialized")
				fmt.Println("💡 Run 'syncstation init' to get started")
				return nil
			}

			localConfig, err := config.LoadLocalConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("🔧 Syncstation Configuration\n\n")
			fmt.Printf("📁 Config Directory: %s\n", getConfigDir())
			fmt.Printf("☁️  Cloud Directory: %s\n", localConfig.CloudSyncDir)
			fmt.Printf("💻 Computer ID: %s\n", localConfig.CurrentComputer)
			fmt.Printf("🔄 Git Mode: %v\n", localConfig.GitMode)
			if localConfig.GitMode && localConfig.GitRepoRoot != "" {
				fmt.Printf("📂 Git Repository: %s\n", localConfig.GitRepoRoot)
			}

			// Show sync files
			fmt.Printf("\n📄 Data Files:\n")
			fmt.Printf("   Items: %s\n", localConfig.GetSyncItemsPath())
			fmt.Printf("   Metadata: %s\n", localConfig.GetFileMetadataPath())
			fmt.Printf("   Configs: %s\n", localConfig.GetCloudConfigsPath())

			return nil
		},
	}

	return cmd
}

// Helper functions

func checkForConflicts(localConfig *config.LocalConfig, items []*config.SyncItem) ([]string, error) {
	var conflicts []string
	diffEngine := diff.NewDiffEngine()

	for _, item := range items {
		localPath := item.GetCurrentComputerPath(localConfig.CurrentComputer)
		if localPath == "" {
			continue // Skip items without paths configured for this computer
		}

		cloudPath := item.GetCloudPath(localConfig.GetCloudConfigsPath())

		// Both must exist to have a conflict
		if !config.PathExists(localPath) || !config.PathExists(cloudPath) {
			continue
		}

		// For files, check if they differ and both have been modified
		if item.Type == "file" {
			fileDiff, err := diffEngine.CompareFiles(localPath, cloudPath)
			if err != nil {
				continue // Skip files we can't compare
			}

			// Check if it's a conflict (both modified since last known sync)
			if fileDiff.Status == "conflict" ||
				(fileDiff.Status == "local_newer" && hasCloudChangedSinceLastSync(localConfig, item.Name, localPath, cloudPath)) ||
				(fileDiff.Status == "cloud_newer" && hasLocalChangedSinceLastSync(localConfig, item.Name, localPath)) {
				conflicts = append(conflicts, item.Name)
			}
		} else {
			// For directories, do a simple timestamp check
			localInfo, err1 := os.Stat(localPath)
			cloudInfo, err2 := os.Stat(cloudPath)
			if err1 == nil && err2 == nil {
				// If both directories have been modified recently, consider it a potential conflict
				if !localInfo.ModTime().Equal(cloudInfo.ModTime()) {
					conflicts = append(conflicts, item.Name+" (directory - manual check recommended)")
				}
			}
		}
	}

	return conflicts, nil
}

func hasCloudChangedSinceLastSync(localConfig *config.LocalConfig, itemName, localPath, cloudPath string) bool {
	// Load cloud metadata to check if cloud file changed since last sync
	cloudMetadata, err := config.LoadFileMetadataDataGitAware(localConfig, localConfig.GetFileMetadataPath())
	if err != nil {
		return true // Assume conflict if we can't load metadata
	}

	if itemMetadata, exists := cloudMetadata.Metadata[itemName]; exists {
		if fileMetadata, exists := itemMetadata[localPath]; exists {
			// Compare current cloud hash with last known cloud hash
			currentCloudHash, err := config.CalculateFileHash(cloudPath)
			if err != nil {
				return true // Assume conflict if we can't calculate hash
			}
			return fileMetadata.CloudHash != currentCloudHash
		}
	}

	return true // No previous sync data, assume conflict
}

func hasLocalChangedSinceLastSync(localConfig *config.LocalConfig, itemName, localPath string) bool {
	// Load local file states to check if local file changed since last sync
	fileStatesPath := filepath.Join(getConfigDir(), "file-states.json")
	fileStates, err := config.LoadFileStatesData(fileStatesPath)
	if err != nil {
		return true // Assume conflict if we can't load states
	}

	if fileState := fileStates.GetFileState(itemName, localPath); fileState != nil {
		currentLocalHash, err := config.CalculateFileHash(localPath)
		if err != nil {
			return true // Assume conflict if we can't calculate hash
		}
		return fileState.LocalHash != currentLocalHash
	}

	return true // No previous sync data, assume conflict
}

func getComputerList(paths map[string]string) []string {
	var computers []string
	for computerID := range paths {
		computers = append(computers, computerID)
	}
	return computers
}

func deleteCloudFiles(cloudPath, itemType string) error {
	if itemType == "file" {
		return os.Remove(cloudPath)
	} else {
		return os.RemoveAll(cloudPath)
	}
}

func cleanupItemMetadata(localConfig *config.LocalConfig, item *config.SyncItem) error {
	// Load cloud metadata
	cloudMetadata, err := config.LoadFileMetadataDataGitAware(localConfig, localConfig.GetFileMetadataPath())
	if err != nil {
		return fmt.Errorf("failed to load cloud metadata: %w", err)
	}

	// Remove metadata for this item
	delete(cloudMetadata.Metadata, item.Name)

	// Save updated metadata
	if err := cloudMetadata.SaveFileMetadataDataGitAware(localConfig, localConfig.GetFileMetadataPath()); err != nil {
		return fmt.Errorf("failed to save updated metadata: %w", err)
	}

	// Load and clean up local file states
	fileStatesPath := filepath.Join(getConfigDir(), "file-states.json")
	fileStates, err := config.LoadFileStatesData(fileStatesPath)
	if err != nil {
		return fmt.Errorf("failed to load file states: %w", err)
	}

	// Remove local states for this item
	delete(fileStates.States, item.Name)

	// Save updated file states
	if err := fileStates.SaveFileStatesData(fileStatesPath); err != nil {
		return fmt.Errorf("failed to save updated file states: %w", err)
	}

	return nil
}

func loadConfig() (*config.LocalConfig, error) {
	configPath := filepath.Join(getConfigDir(), "config.json")
	localConfig, err := config.LoadLocalConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if localConfig.CloudSyncDir == "" {
		return nil, fmt.Errorf("not initialized. Run 'syncstation init' first")
	}

	// Override computer ID if specified
	if computer != "" {
		localConfig.CurrentComputer = computer
	}

	return localConfig, nil
}

func performSyncWithConflictCheck(operation sync.SyncOperation, args []string, force bool) error {
	// Load configuration
	localConfig, err := loadConfig()
	if err != nil {
		return err
	}

	// Load sync items
	syncItems, err := config.LoadSyncItemsData(localConfig.GetSyncItemsPath())
	if err != nil {
		return fmt.Errorf("failed to load sync items: %w", err)
	}

	if len(syncItems.SyncItems) == 0 {
		fmt.Println("📭 No sync items configured")
		return nil
	}

	// Filter items if specific item requested
	itemsToSync := syncItems.SyncItems
	if len(args) > 0 {
		itemName := args[0]
		item := syncItems.FindSyncItem(itemName)
		if item == nil {
			return fmt.Errorf("sync item not found: %s", itemName)
		}
		itemsToSync = []*config.SyncItem{item}
	}

	// Check for conflicts if not forced
	if !force {
		conflicts, err := checkForConflicts(localConfig, itemsToSync)
		if err != nil {
			return fmt.Errorf("failed to check for conflicts: %w", err)
		}

		if len(conflicts) > 0 {
			operationName := "push"
			if operation == sync.SyncPull {
				operationName = "pull"
			}

			fmt.Printf("⚠️  Conflicts detected! The following items have been modified on both local and cloud:\n\n")
			for _, conflict := range conflicts {
				fmt.Printf("   🔥 %s\n", conflict)
			}
			fmt.Printf("\nUsing %s will overwrite changes and may cause data loss.\n", operationName)
			fmt.Printf("💡 Options:\n")
			fmt.Printf("   • Run 'syncstation sync' to see detailed conflict information\n")
			fmt.Printf("   • Use 'syncstation %s --force' to proceed anyway\n", operationName)
			fmt.Printf("   • Resolve conflicts manually first\n")
			return fmt.Errorf("operation cancelled due to conflicts")
		}
	}

	return performSync(operation, args)
}

func performSync(operation sync.SyncOperation, args []string) error {
	// Load configuration
	localConfig, err := loadConfig()
	if err != nil {
		return err
	}

	// Load sync items
	syncItems, err := config.LoadSyncItemsData(localConfig.GetSyncItemsPath())
	if err != nil {
		return fmt.Errorf("failed to load sync items: %w", err)
	}

	if len(syncItems.SyncItems) == 0 {
		fmt.Println("📭 No sync items configured")
		return nil
	}

	// Create sync engine
	diffEngine := diff.NewDiffEngine()
	syncEngine := sync.NewSyncEngine(localConfig, diffEngine)

	// Filter items if specific item requested
	itemsToSync := syncItems.SyncItems
	if len(args) > 0 {
		itemName := args[0]
		item := syncItems.FindSyncItem(itemName)
		if item == nil {
			return fmt.Errorf("sync item not found: %s", itemName)
		}
		itemsToSync = []*config.SyncItem{item}
	}

	// Show operation header
	operationName := "Smart Sync"
	operationIcon := "🔄"
	switch operation {
	case sync.SyncPush:
		operationName = "Push"
		operationIcon = "⬆️"
	case sync.SyncPull:
		operationName = "Pull"
		operationIcon = "⬇️"
	}

	fmt.Printf("%s %s - %d items\n\n", operationIcon, operationName, len(itemsToSync))

	if dryRun {
		fmt.Println("🔍 DRY RUN MODE - No changes will be made\n")
	}

	// Perform sync
	result, err := syncEngine.SyncAll(operation, itemsToSync)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Display results
	if result.Success {
		fmt.Printf("✅ %s\n", result.Message)
	} else {
		fmt.Printf("⚠️  %s\n", result.Message)
	}

	if len(result.Errors) > 0 {
		fmt.Println("\n❌ Errors:")
		for _, errMsg := range result.Errors {
			fmt.Printf("   %s\n", errMsg)
		}
	}

	if verbose {
		fmt.Printf("\n📊 Summary:\n")
		fmt.Printf("   Changed: %d\n", result.FilesChanged)
		fmt.Printf("   Skipped: %d\n", result.FilesSkipped)
		fmt.Printf("   Errors: %d\n", result.FilesErrored)
	}

	return nil
}

func getConfigDir() string {
	if configDir != "" {
		return configDir
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "syncstation")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "syncstation")
	default: // linux, unix
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			return filepath.Join(xdgConfig, "syncstation")
		}
		return filepath.Join(os.Getenv("HOME"), ".config", "syncstation")
	}
}

func getComputerID() string {
	// Try hostname first
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return hostname
	}

	// Fall back to OS and architecture
	return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
}

func isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	return config.PathExists(gitDir)
}

func findGitRoot(path string) string {
	current := path
	for {
		if isGitRepository(current) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return path
}

func getPathStatus(path string) string {
	if path == "" {
		return "❌ Not configured"
	}

	expandedPath := config.ExpandPath(path)
	if !config.PathExists(expandedPath) {
		return fmt.Sprintf("❌ Missing: %s", expandedPath)
	}

	return fmt.Sprintf("✅ %s", expandedPath)
}

func getStatusIcon(status string) string {
	switch status {
	case "same":
		return "✅ In sync"
	case "local_newer":
		return "⬆️  Local newer"
	case "cloud_newer":
		return "⬇️  Cloud newer"
	case "conflict":
		return "⚠️  Conflict (same timestamp, different content)"
	case "local_only":
		return "⬆️  Local only - need to push"
	case "cloud_only":
		return "⬇️  Cloud only - need to pull"
	default:
		return "❓ Unknown status"
	}
}
