package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
	
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// App represents the main application
type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
	config     *Config
	diffEngine *DiffEngine
	syncEngine *SyncEngine
	
	// UI components
	leftPanel   *fyne.Container
	rightPanel  *fyne.Container
	syncTree    *widget.Tree
	diffViewer  *widget.RichText
	statusBar   *widget.Label
	
	// State
	selectedSyncItemIndex int // Index of currently selected sync item (-1 if none)
}

// NewApp creates a new application instance
func NewApp() *App {
	fyneApp := app.New()
	fyneApp.SetIcon(theme.ComputerIcon())
	
	mainWindow := fyneApp.NewWindow("Config Sync Tool")
	mainWindow.SetIcon(theme.ComputerIcon())
	
	// Load configuration
	config, err := LoadConfig("config.json")
	if err != nil {
		config = NewConfig()
	}
	
	// Detect current computer
	config.DetectCurrentComputer()
	
	// No default sync items - user will add them manually
	
	diffEngine := NewDiffEngine(config)
	syncEngine := NewSyncEngine(config, diffEngine)
	
	return &App{
		fyneApp:    fyneApp,
		mainWindow: mainWindow,
		config:     config,
		diffEngine: diffEngine,
		syncEngine: syncEngine,
		selectedSyncItemIndex: -1,
	}
}

// Run starts the application
func (a *App) Run() {
	a.setupUI()
	a.mainWindow.Resize(fyne.NewSize(1200, 800))
	
	// Save config on window close
	a.mainWindow.SetCloseIntercept(func() {
		a.config.SaveConfig("config.json")
		a.mainWindow.Close()
	})
	
	a.mainWindow.ShowAndRun()
}

// setupUI creates the main UI layout
func (a *App) setupUI() {
	// Create left panel components
	a.createLeftPanel()
	
	// Create right panel components
	a.createRightPanel()
	
	// Create main split container
	mainSplit := container.NewHSplit(
		a.leftPanel,
		a.rightPanel,
	)
	mainSplit.SetOffset(0.3) // 30% for left panel, 70% for right panel
	
	// Create modern status bar with icon
	statusIcon := widget.NewIcon(theme.InfoIcon())
	a.statusBar = widget.NewLabel("Ready")
	statusBar := container.NewHBox(
		statusIcon,
		a.statusBar,
	)
	
	// Create main layout with better spacing and padding
	content := container.NewBorder(
		nil,           // top
		container.NewPadded(container.NewVBox(
			widget.NewSeparator(),
			statusBar,
		)), // bottom with padding
		nil,           // left
		nil,           // right
		mainSplit,     // center
	)
	
	a.mainWindow.SetContent(content)
}

// createLeftPanel creates the left panel with sync items tree
func (a *App) createLeftPanel() {
	// Create tree widget for sync items
	a.syncTree = widget.NewTree(
		a.treeChildUIDs,
		a.treeIsBranch,
		a.treeCreateNode,
		a.treeUpdateNode,
	)
	
	// Add selection handler for tree
	a.syncTree.OnSelected = func(uid string) {
		a.onTreeItemSelected(uid)
	}
	
	// Create modern buttons with icons
	syncAllBtn := widget.NewButtonWithIcon("Smart Sync", theme.ViewRefreshIcon(), func() {
		a.syncAll(SyncSmart)
	})
	syncAllBtn.Importance = widget.HighImportance
	
	pushBtn := widget.NewButtonWithIcon("Push All", theme.MailSendIcon(), func() {
		a.syncAll(SyncPush)
	})
	pushBtn.Importance = widget.MediumImportance
	
	pullBtn := widget.NewButtonWithIcon("Pull All", theme.DownloadIcon(), func() {
		a.syncAll(SyncPull)
	})
	pullBtn.Importance = widget.MediumImportance
	
	addItemBtn := widget.NewButtonWithIcon("Add Item", theme.ContentAddIcon(), func() {
		a.showAddItemDialog()
	})
	
	editItemBtn := widget.NewButtonWithIcon("Edit Selected", theme.DocumentCreateIcon(), func() {
		if a.selectedSyncItemIndex >= 0 && a.selectedSyncItemIndex < len(a.config.SyncItems) {
			a.showEditItemDialog()
		} else {
			dialog.ShowInformation("No Selection", "Please select a sync item from the tree to edit.", a.mainWindow)
		}
	})
	
	deleteItemBtn := widget.NewButtonWithIcon("Delete Selected", theme.DeleteIcon(), func() {
		if a.selectedSyncItemIndex >= 0 && a.selectedSyncItemIndex < len(a.config.SyncItems) {
			a.showDeleteItemDialog()
		} else {
			dialog.ShowInformation("No Selection", "Please select a sync item from the tree to delete.", a.mainWindow)
		}
	})
	deleteItemBtn.Importance = widget.DangerImportance
	
	settingsBtn := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() {
		a.showSettingsDialog()
	})
	
	// Create modern button container with increased spacing
	buttonContainer := container.NewVBox(
		widget.NewCard("Sync Operations", "", container.NewVBox(
			syncAllBtn,
			widget.NewSeparator(),
			container.NewHBox(pushBtn, pullBtn),
		)),
		widget.NewSeparator(),
		widget.NewCard("Management", "", container.NewVBox(
			addItemBtn,
			widget.NewSeparator(),
			editItemBtn,
			deleteItemBtn,
			widget.NewSeparator(),
			settingsBtn,
		)),
	)
	
	// Create left panel layout with padding
	a.leftPanel = container.NewBorder(
		nil,              // top
		container.NewPadded(buttonContainer),  // bottom with padding
		nil,              // left
		nil,              // right
		container.NewPadded(container.NewScroll(a.syncTree)), // center with padding
	)
}

// createRightPanel creates the right panel for diff viewing
func (a *App) createRightPanel() {
	// Create diff viewer
	a.diffViewer = widget.NewRichText()
	a.diffViewer.ParseMarkdown("# Welcome to Config Sync Tool\n\nSelect a file from the left panel to view differences between local and cloud versions.\n\n**How to use:**\n1. Add sync items using the \"Add Item\" button\n2. Select a sync item from the tree (bold names are editable)\n3. Use \"Edit Selected\" or \"Delete Selected\" buttons to manage items\n4. Click on files to view diffs\n\n**Features:**\n- Smart sync with conflict detection\n- Visual diff viewer\n- Cross-platform support\n- Cloud storage agnostic")
	
	// Create header with icon
	header := container.NewHBox(
		widget.NewIcon(theme.DocumentIcon()),
		widget.NewLabelWithStyle("Diff Viewer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	
	// Create right panel layout with modern styling and padding
	a.rightPanel = container.NewBorder(
		container.NewPadded(container.NewVBox(
			header,
			widget.NewSeparator(),
		)), // top with padding
		nil,                            // bottom
		nil,                            // left
		nil,                            // right
		container.NewPadded(container.NewScroll(a.diffViewer)), // center with padding
	)
}

// TreeNode represents a node in the file tree
type TreeNode struct {
	ID       string
	Name     string
	FullPath string
	IsDir    bool
	IsRoot   bool
	ItemIndex int
	Children map[string]*TreeNode
}

// buildFileTree builds a nested tree structure from file paths
func (a *App) buildFileTree(itemIndex int, files []string) *TreeNode {
	root := &TreeNode{
		ID:       fmt.Sprintf("item_%d", itemIndex),
		Name:     a.config.SyncItems[itemIndex].Name,
		IsDir:    true,
		IsRoot:   true,
		ItemIndex: itemIndex,
		Children: make(map[string]*TreeNode),
	}
	
	// Sort files for consistent ordering
	sort.Strings(files)
	
	for _, file := range files {
		if file == "" {
			continue
		}
		
		parts := strings.Split(file, "/")
		current := root
		currentPath := ""
		
		for i, part := range parts {
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}
			
			isFile := i == len(parts)-1
			nodeID := fmt.Sprintf("item_%d_%s", itemIndex, strings.ReplaceAll(currentPath, "/", "___"))
			
			if _, exists := current.Children[part]; !exists {
				current.Children[part] = &TreeNode{
					ID:       nodeID,
					Name:     part,
					FullPath: currentPath,
					IsDir:    !isFile,
					IsRoot:   false,
					ItemIndex: itemIndex,
					Children: make(map[string]*TreeNode),
				}
			}
			current = current.Children[part]
		}
	}
	
	return root
}

// Tree data source methods
func (a *App) treeChildUIDs(uid string) []string {
	if uid == "" {
		// Root level - return sync items
		var items []string
		for i := range a.config.SyncItems {
			items = append(items, fmt.Sprintf("item_%d", i))
		}
		sort.Strings(items) // Sort root items
		return items
	}
	
	// Parse the UID to understand what we're dealing with
	if strings.HasPrefix(uid, "item_") {
		parts := strings.Split(uid, "___")
		baseItemID := parts[0] // "item_X"
		
		// Extract item index
		itemIndex := 0
		fmt.Sscanf(baseItemID, "item_%d", &itemIndex)
		
		if itemIndex >= len(a.config.SyncItems) {
			return []string{}
		}
		
		syncItem := a.config.SyncItems[itemIndex]
		
		// Get all files from both local and cloud paths
		allFiles := make(map[string]bool)
		
		// Check local path
		localPath := a.config.GetCurrentComputerPath(syncItem)
		if localPath != "" {
			localFiles, err := a.diffEngine.getFilesInDirectory(localPath)
			if err == nil {
				for _, file := range localFiles {
					allFiles[file] = true
				}
			}
		}
		
		// Check cloud path
		cloudPath := a.config.GetCloudPath(syncItem)
		cloudFiles, err := a.diffEngine.getFilesInDirectory(cloudPath)
		if err == nil {
			for _, file := range cloudFiles {
				allFiles[file] = true
			}
		}
		
		// Convert to sorted slice
		var files []string
		for file := range allFiles {
			if file != "" {
				files = append(files, file)
			}
		}
		
		// Build the tree
		tree := a.buildFileTree(itemIndex, files)
		
		// Navigate to the requested node
		currentNode := tree
		if len(parts) > 1 {
			// Navigate to the specific folder
			pathParts := strings.Split(parts[1], "/")
			for _, part := range pathParts {
				if part != "" {
					if child, exists := currentNode.Children[part]; exists {
						currentNode = child
					} else {
						return []string{} // Path not found
					}
				}
			}
		}
		
		// Return children of current node, sorted by name
		var result []string
		var childNames []string
		for name := range currentNode.Children {
			childNames = append(childNames, name)
		}
		sort.Strings(childNames)
		
		for _, name := range childNames {
			child := currentNode.Children[name]
			result = append(result, child.ID)
		}
		
		return result
	}
	
	return []string{}
}

func (a *App) treeIsBranch(uid string) bool {
	if uid == "" {
		return true // Root is a branch
	}
	
	if !strings.HasPrefix(uid, "item_") {
		return false
	}
	
	// Parse the node to determine if it's a directory
	parts := strings.Split(uid, "___")
	baseItemID := parts[0] // "item_X"
	
	// Extract item index
	itemIndex := 0
	fmt.Sscanf(baseItemID, "item_%d", &itemIndex)
	
	if itemIndex >= len(a.config.SyncItems) {
		return false
	}
	
	// Root sync items are always branches
	if len(parts) == 1 {
		return true
	}
	
	// For nested items, we need to check if it's a directory
	// We'll determine this by checking if it has children in our file structure
	syncItem := a.config.SyncItems[itemIndex]
	
	// Get all files
	allFiles := make(map[string]bool)
	
	localPath := a.config.GetCurrentComputerPath(syncItem)
	if localPath != "" {
		localFiles, err := a.diffEngine.getFilesInDirectory(localPath)
		if err == nil {
			for _, file := range localFiles {
				allFiles[file] = true
			}
		}
	}
	
	cloudPath := a.config.GetCloudPath(syncItem)
	cloudFiles, err := a.diffEngine.getFilesInDirectory(cloudPath)
	if err == nil {
		for _, file := range cloudFiles {
			allFiles[file] = true
		}
	}
	
	// Convert to slice and build tree
	var files []string
	for file := range allFiles {
		if file != "" {
			files = append(files, file)
		}
	}
	
	tree := a.buildFileTree(itemIndex, files)
	
	// Navigate to the requested node
	currentNode := tree
	if len(parts) > 1 {
		pathParts := strings.Split(parts[1], "/")
		for _, part := range pathParts {
			if part != "" {
				if child, exists := currentNode.Children[part]; exists {
					currentNode = child
				} else {
					return false // Path not found, assume it's a file
				}
			}
		}
	}
	
	return currentNode.IsDir
}

func (a *App) treeCreateNode(branch bool) fyne.CanvasObject {
	icon := widget.NewIcon(theme.FolderIcon())
	label := widget.NewLabel("Branch")
	
	// Create a container that will hold the content
	nodeContainer := container.NewHBox(icon, label)
	
	return nodeContainer
}

func (a *App) treeUpdateNode(uid string, branch bool, node fyne.CanvasObject) {
	cont := node.(*fyne.Container)
	icon := cont.Objects[0].(*widget.Icon)
	label := cont.Objects[1].(*widget.Label)
	
	if uid == "" {
		// Root node
		label.SetText("Sync Items")
		icon.SetResource(theme.ComputerIcon())
		return
	}
	
	// Parse the UID to get node information
	parts := strings.Split(uid, "___")
	baseItemID := parts[0] // "item_X"
	
	itemIndex := 0
	fmt.Sscanf(baseItemID, "item_%d", &itemIndex)
	
	if itemIndex >= len(a.config.SyncItems) {
		return
	}
	
	if branch {
		// This is a directory or root sync item
		isRootItem := len(parts) == 1
		if isRootItem {
			// Root sync item
			syncItem := a.config.SyncItems[itemIndex]
			label.SetText(syncItem.Name)
			icon.SetResource(theme.FolderIcon())
			
			// Add additional visual cue for root items that can be edited
			label.TextStyle.Bold = true
		} else {
			// Nested directory
			pathPart := strings.ReplaceAll(parts[1], "___", "/")
			dirName := filepath.Base(pathPart)
			label.SetText(dirName)
			icon.SetResource(theme.FolderIcon())
		}
	} else {
		// This is a file
		if len(parts) > 1 {
			fileName := strings.ReplaceAll(parts[1], "___", "/")
			displayName := filepath.Base(fileName)
			label.SetText(displayName)
			
			// Set icon based on file extension
			ext := strings.ToLower(filepath.Ext(displayName))
			switch ext {
			case ".json":
				icon.SetResource(theme.DocumentIcon())
			case ".yaml", ".yml":
				icon.SetResource(theme.DocumentIcon())
			case ".md":
				icon.SetResource(theme.DocumentIcon())
			case ".txt":
				icon.SetResource(theme.DocumentIcon())
			case ".toml":
				icon.SetResource(theme.DocumentIcon())
			case ".xml":
				icon.SetResource(theme.DocumentIcon())
			case ".ini", ".conf", ".cfg":
				icon.SetResource(theme.DocumentIcon())
			case ".log":
				icon.SetResource(theme.DocumentIcon())
			default:
				icon.SetResource(theme.FileIcon())
			}
		}
	}
}

// Event handlers
func (a *App) onTreeItemSelected(uid string) {
	if uid == "" {
		a.selectedSyncItemIndex = -1
		return
	}
	
	// Check if this is a file (leaf node)
	if !a.treeIsBranch(uid) {
		// This is a file, show diff
		a.showFileDiff(uid)
	} else {
		// This is a sync item, show its status and track selection
		parts := strings.Split(uid, "_")
		if len(parts) == 2 && parts[0] == "item" {
			itemIndex := 0
			fmt.Sscanf(parts[1], "%d", &itemIndex)
			a.selectedSyncItemIndex = itemIndex
		}
		a.showSyncItemStatus(uid)
	}
}

func (a *App) showFileDiff(uid string) {
	// Parse the file UID to get sync item and file info
	// Format: "item_X___path/to/file" where X is sync item index
	parts := strings.Split(uid, "___")
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "item_") {
		a.diffViewer.ParseMarkdown("**Error:** Invalid file ID")
		return
	}
	
	baseItemID := parts[0] // "item_X"
	itemIndex := 0
	fmt.Sscanf(baseItemID, "item_%d", &itemIndex)
	
	if itemIndex >= len(a.config.SyncItems) {
		a.diffViewer.ParseMarkdown("**Error:** Sync item not found")
		return
	}
	
	syncItem := a.config.SyncItems[itemIndex]
	// Reconstruct the filename from the UID
	fileName := strings.ReplaceAll(parts[1], "___", "/")
	
	// Get the file paths
	localPath := a.config.GetCurrentComputerPath(syncItem)
	cloudPath := a.config.GetCloudPath(syncItem)
	
	if localPath == "" {
		a.diffViewer.ParseMarkdown("**Error:** No local path configured for current computer")
		return
	}
	
	localFilePath := filepath.Join(localPath, fileName)
	cloudFilePath := filepath.Join(cloudPath, fileName)
	
	// Generate diff
	diff, err := a.diffEngine.CompareFiles(localFilePath, cloudFilePath)
	if err != nil {
		a.diffViewer.ParseMarkdown("**Error:** " + err.Error())
		return
	}
	
	a.displayFileDiff(diff)
	a.statusBar.SetText("Showing diff for " + filepath.Base(fileName))
}

func (a *App) displayFileDiff(diff *FileDiff) {
	var content strings.Builder
	
	content.WriteString("# File Diff\n\n")
	content.WriteString(fmt.Sprintf("**Local:** %s\n", diff.LocalPath))
	content.WriteString(fmt.Sprintf("**Cloud:** %s\n", diff.CloudPath))
	content.WriteString(fmt.Sprintf("**Status:** %s\n\n", diff.Status))
	
	if diff.LocalExists {
		content.WriteString(fmt.Sprintf("**Local Modified:** %s\n", diff.LocalModTime.Format("2006-01-02 15:04:05")))
	}
	if diff.CloudExists {
		content.WriteString(fmt.Sprintf("**Cloud Modified:** %s\n", diff.CloudModTime.Format("2006-01-02 15:04:05")))
	}
	
	content.WriteString("\n---\n\n")
	
	switch diff.Status {
	case "same":
		content.WriteString("✅ **Files are identical**")
	case "local_only":
		content.WriteString("📄 **File exists only locally**")
	case "cloud_only":
		content.WriteString("☁️ **File exists only in cloud**")
	case "local_newer":
		content.WriteString("🔄 **Local file is newer**")
	case "cloud_newer":
		content.WriteString("🔄 **Cloud file is newer**")
	case "conflict":
		content.WriteString("⚠️ **Conflict detected**")
	default:
		content.WriteString("❓ **Unknown status**")
	}
	
	// Show line diff if available
	if len(diff.Lines) > 0 {
		content.WriteString("\n\n## Line-by-line Diff\n\n")
		for _, line := range diff.Lines {
			switch line.Type {
			case "same":
				content.WriteString(fmt.Sprintf("  %d: %s\n", line.LineNumber, line.Content))
			case "added":
				content.WriteString(fmt.Sprintf("+ %d: %s\n", line.LineNumber, line.Content))
			case "removed":
				content.WriteString(fmt.Sprintf("- %d: %s\n", line.LineNumber, line.Content))
			}
		}
	}
	
	a.diffViewer.ParseMarkdown(content.String())
}

func (a *App) showSyncItemStatus(uid string) {
	// Show general info about the sync item
	a.diffViewer.ParseMarkdown("**Sync Item:** " + uid + "\n\n*Select a file to view differences.*")
	a.statusBar.SetText("Selected sync item: " + uid)
}

func (a *App) syncAll(operation SyncOperation) {
	operationName := "Smart Sync"
	switch operation {
	case SyncPush:
		operationName = "Push"
	case SyncPull:
		operationName = "Pull"
	}
	
	a.statusBar.SetText(fmt.Sprintf("Performing %s...", operationName))
	
	// Perform sync operation
	result, err := a.syncEngine.SyncAll(operation)
	if err != nil {
		a.statusBar.SetText(fmt.Sprintf("Sync failed: %v", err))
		return
	}
	
	// Update status bar with results
	statusMsg := fmt.Sprintf("%s completed: %d changed, %d skipped", 
		operationName, result.FilesChanged, result.FilesSkipped)
	
	if result.FilesErrored > 0 {
		statusMsg += fmt.Sprintf(", %d errors", result.FilesErrored)
	}
	
	a.statusBar.SetText(statusMsg)
	
	// Save configuration
	a.config.LastSync = time.Now()
	if err := a.config.SaveConfig("config.json"); err != nil {
		a.statusBar.SetText("Sync completed but failed to save config")
	}
}

func (a *App) showAddItemDialog() {
	// Create entry widgets
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("e.g., Claude Config, VS Code Settings")
	
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("Select folder path...")
	
	// Create unified browse button for both files and folders
	browseBtn := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		a.showUnifiedBrowser(pathEntry)
	})
	
	// Create form layout with padding
	form := container.NewVBox(
		widget.NewCard("Add New Sync Item", "", container.NewPadded(container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("Name", nameEntry),
				widget.NewFormItem("Path for "+a.config.CurrentComputer, container.NewBorder(
					nil, nil, nil, browseBtn, pathEntry,
				)),
			),
		))),
	)
	
	// Create dialog
	addDialog := dialog.NewCustom("Add Sync Item", "", form, a.mainWindow)
	
	// Add buttons
	addBtn := widget.NewButtonWithIcon("Add Item", theme.ConfirmIcon(), func() {
		name := strings.TrimSpace(nameEntry.Text)
		path := strings.TrimSpace(pathEntry.Text)
		
		if name == "" {
			dialog.ShowError(fmt.Errorf("please enter a name for the sync item"), a.mainWindow)
			return
		}
		
		if path == "" {
			dialog.ShowError(fmt.Errorf("please select a path"), a.mainWindow)
			return
		}
		
		// Check if item already exists
		for _, item := range a.config.SyncItems {
			if item.Name == name {
				dialog.ShowError(fmt.Errorf("sync item with name '%s' already exists", name), a.mainWindow)
				return
			}
		}
		
		a.config.AddSyncItem(name, map[string]string{
			a.config.CurrentComputer: path,
		})
		a.statusBar.SetText("Added sync item: " + name)
		addDialog.Hide()
		
		// Refresh tree view to show new item
		a.syncTree.Refresh()
	})
	addBtn.Importance = widget.HighImportance
	
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		addDialog.Hide()
	})
	
	buttons := container.NewHBox(addBtn, cancelBtn)
	form.Add(container.NewPadded(buttons))
	
	addDialog.Resize(fyne.NewSize(500, 250))
	addDialog.Show()
}

func (a *App) showEditItemDialog() {
	// Check if a sync item is selected
	if a.selectedSyncItemIndex < 0 || a.selectedSyncItemIndex >= len(a.config.SyncItems) {
		dialog.ShowError(fmt.Errorf("please select a sync item to edit"), a.mainWindow)
		return
	}
	
	selectedItem := a.config.SyncItems[a.selectedSyncItemIndex]
	
	// Create entry widgets with current values
	nameEntry := widget.NewEntry()
	nameEntry.SetText(selectedItem.Name)
	nameEntry.SetPlaceHolder("e.g., Claude Config, VS Code Settings")
	
	pathEntry := widget.NewEntry()
	currentPath := a.config.GetCurrentComputerPath(selectedItem)
	pathEntry.SetText(currentPath)
	pathEntry.SetPlaceHolder("Select folder/file path...")
	
	// Create unified browse button for both files and folders
	browseBtn := widget.NewButtonWithIcon("Browse", theme.FolderOpenIcon(), func() {
		a.showUnifiedBrowser(pathEntry)
	})
	
	// Create form layout
	form := container.NewVBox(
		widget.NewCard("Edit Sync Item", "", container.NewPadded(container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("Name", nameEntry),
				widget.NewFormItem("Path for "+a.config.CurrentComputer, container.NewBorder(
					nil, nil, nil, browseBtn, pathEntry,
				)),
			),
		))),
	)
	
	// Create dialog
	editDialog := dialog.NewCustom("Edit Sync Item", "", form, a.mainWindow)
	
	// Add buttons
	saveBtn := widget.NewButtonWithIcon("Save Changes", theme.DocumentSaveIcon(), func() {
		name := strings.TrimSpace(nameEntry.Text)
		path := strings.TrimSpace(pathEntry.Text)
		
		if name == "" {
			dialog.ShowError(fmt.Errorf("please enter a name for the sync item"), a.mainWindow)
			return
		}
		
		if path == "" {
			dialog.ShowError(fmt.Errorf("please select a path"), a.mainWindow)
			return
		}
		
		// Check if name conflicts with other items (excluding current)
		for i, item := range a.config.SyncItems {
			if i != a.selectedSyncItemIndex && item.Name == name {
				dialog.ShowError(fmt.Errorf("sync item with name '%s' already exists", name), a.mainWindow)
				return
			}
		}
		
		// Update the sync item
		selectedItem.Name = name
		if selectedItem.Paths == nil {
			selectedItem.Paths = make(map[string]string)
		}
		selectedItem.Paths[a.config.CurrentComputer] = path
		
		a.statusBar.SetText("Updated sync item: " + name)
		editDialog.Hide()
		
		// Refresh tree view
		a.syncTree.Refresh()
	})
	saveBtn.Importance = widget.HighImportance
	
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		// Confirm deletion
		confirmDialog := dialog.NewConfirm("Delete Sync Item", 
			fmt.Sprintf("Are you sure you want to delete '%s'?", selectedItem.Name),
			func(confirmed bool) {
				if confirmed {
					// Remove the item from the slice
					a.config.SyncItems = append(a.config.SyncItems[:a.selectedSyncItemIndex], 
						a.config.SyncItems[a.selectedSyncItemIndex+1:]...)
					a.selectedSyncItemIndex = -1
					a.statusBar.SetText("Deleted sync item")
					editDialog.Hide()
					a.syncTree.Refresh()
				}
			}, a.mainWindow)
		confirmDialog.Show()
	})
	deleteBtn.Importance = widget.DangerImportance
	
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		editDialog.Hide()
	})
	
	buttons := container.NewHBox(saveBtn, deleteBtn, cancelBtn)
	form.Add(container.NewPadded(buttons))
	
	editDialog.Resize(fyne.NewSize(500, 300))
	editDialog.Show()
}

func (a *App) showDeleteItemDialog() {
	// Check if a sync item is selected
	if a.selectedSyncItemIndex < 0 || a.selectedSyncItemIndex >= len(a.config.SyncItems) {
		dialog.ShowError(fmt.Errorf("please select a sync item to delete"), a.mainWindow)
		return
	}
	
	selectedItem := a.config.SyncItems[a.selectedSyncItemIndex]
	
	// Show confirmation dialog
	confirmDialog := dialog.NewConfirm("Delete Sync Item", 
		fmt.Sprintf("Are you sure you want to delete '%s'?\n\nThis will remove the sync item from your configuration but will not delete the actual files.", selectedItem.Name),
		func(confirmed bool) {
			if confirmed {
				// Remove the item from the slice
				a.config.SyncItems = append(a.config.SyncItems[:a.selectedSyncItemIndex], 
					a.config.SyncItems[a.selectedSyncItemIndex+1:]...)
				a.selectedSyncItemIndex = -1
				a.statusBar.SetText("Deleted sync item: " + selectedItem.Name)
				a.syncTree.Refresh()
				
				// Save config
				a.config.SaveConfig("config.json")
			}
		}, a.mainWindow)
	confirmDialog.Show()
}

func (a *App) showUnifiedBrowser(pathEntry *widget.Entry) {
	// Create a unified browser that allows selection of both files and folders
	// and shows hidden files by default
	folderDialog := dialog.NewFolderOpen(func(folder fyne.ListableURI, err error) {
		if err != nil {
			return
		}
		if folder != nil {
			pathEntry.SetText(folder.Path())
		}
	}, a.mainWindow)
	
	// Show hidden files by default
	folderDialog.SetFilter(nil) // No filter to show all files including hidden ones
	folderDialog.Show()
	
	// Note: Fyne's built-in dialogs have limitations for unified file/folder selection
	// This implementation focuses on folder selection which is the primary use case
	// Users can manually type file paths if needed
}

func (a *App) showSettingsDialog() {
	// Create computer selection
	var computerIDs []string
	var computerNames []string
	for id, computer := range a.config.Computers {
		computerIDs = append(computerIDs, id)
		computerNames = append(computerNames, computer.Name)
	}
	
	computerSelect := widget.NewSelect(computerNames, func(selected string) {
		// Find the ID for the selected name
		for id, computer := range a.config.Computers {
			if computer.Name == selected {
				a.config.CurrentComputer = id
				break
			}
		}
	})
	
	// Set current selection
	if currentComp, exists := a.config.Computers[a.config.CurrentComputer]; exists {
		computerSelect.SetSelected(currentComp.Name)
	}
	
	// Create form
	form := widget.NewForm(
		widget.NewFormItem("Current Computer", computerSelect),
		widget.NewFormItem("Config Path", widget.NewLabel(a.config.ConfigsPath)),
		widget.NewFormItem("Last Sync", widget.NewLabel(a.config.LastSync.Format("2006-01-02 15:04:05"))),
	)
	
	// Create settings card with padding
	settingsCard := widget.NewCard("Application Settings", "", container.NewPadded(form))
	
	// Create computer info with padding
	computerInfo := widget.NewCard("Computer Information", "", container.NewPadded(container.NewVBox(
		widget.NewLabel("Hostname: " + a.config.CurrentComputer),
		widget.NewLabel("Sync Items: " + fmt.Sprintf("%d", len(a.config.SyncItems))),
		widget.NewLabel("Computers: " + fmt.Sprintf("%d", len(a.config.Computers))),
	)))
	
	content := container.NewVBox(settingsCard, computerInfo)
	
	// Create dialog
	settingsDialog := dialog.NewCustom("Settings", "", content, a.mainWindow)
	
	// Add buttons
	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		err := a.config.SaveConfig("config.json")
		if err != nil {
			dialog.ShowError(err, a.mainWindow)
		} else {
			a.statusBar.SetText("Settings saved")
		}
		settingsDialog.Hide()
	})
	saveBtn.Importance = widget.HighImportance
	
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		settingsDialog.Hide()
	})
	
	buttons := container.NewHBox(saveBtn, cancelBtn)
	content.Add(container.NewPadded(buttons))
	
	settingsDialog.Resize(fyne.NewSize(450, 350))
	settingsDialog.Show()
}

func main() {
	app := NewApp()
	app.Run()
}