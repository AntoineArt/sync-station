package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AntoineArt/syncstation/internal/config"
)

// TUI styles - Enhanced with fancy visual elements
var (
	// Main container with fancy border
	mainBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2)

	// Title with gradient-like effect and emoji
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#7D56F4"}).
		Padding(0, 2).
		MarginBottom(1).
		Width(60).
		Align(lipgloss.Center)

	// Header info bar with icons
	headerInfoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#BD93F9")).
		Background(lipgloss.Color("#282A36")).
		Padding(0, 2).
		MarginBottom(1).
		Width(60).
		Align(lipgloss.Center)

	// Content area with inner border
	contentBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6272A4")).
		Padding(1, 2).
		MarginBottom(1)

	// Items list header
	itemsHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#8BE9FD")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("#6272A4")).
		MarginBottom(1).
		PaddingBottom(1)

	// Enhanced item styling
	itemStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		MarginBottom(0)

	selectedItemStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("#FF79C6")).
		Background(lipgloss.Color("#44475A")).
		Bold(true)

	// Fancy status indicators with colors
	syncedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#50FA7B")).
		Bold(true)

	warningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFB86C")).
		Bold(true)

	conflictStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555")).
		Bold(true)

	localNewerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8BE9FD")).
		Bold(true)

	cloudNewerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#BD93F9")).
		Bold(true)

	// Footer help bar
	helpBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8F8F2")).
		Background(lipgloss.Color("#44475A")).
		Padding(0, 2).
		Width(60).
		Align(lipgloss.Center)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5555")).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5555")).
		Padding(1, 2)

	// Secondary text styling
	dimmedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4"))

	pathStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1FA8C")).
		Italic(true)

	fileCountStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8BE9FD"))
)

// TUI model
type tuiModel struct {
	localConfig   *config.LocalConfig
	syncItems     *config.SyncItemsData
	cursor        int
	selected      map[int]bool
	showStatus    bool
	lastStatus    string
	width         int
	height        int
	err           error
}

type statusMsg struct {
	message string
	isError bool
}

// InitialTUIModel initializes the TUI model
func InitialTUIModel() (tuiModel, error) {
	// Load local config
	localConfig, err := config.LoadLocalConfig(filepath.Join(getConfigDir(), "config.json"))
	if err != nil {
		return tuiModel{}, fmt.Errorf("failed to load config: %w", err)
	}

	if localConfig.CloudSyncDir == "" {
		return tuiModel{}, fmt.Errorf("not initialized. Run 'syncstation init' first")
	}

	// Load sync items
	syncItems, err := config.LoadSyncItemsData(localConfig.GetSyncItemsPath())
	if err != nil {
		return tuiModel{}, fmt.Errorf("failed to load sync items: %w", err)
	}

	return tuiModel{
		localConfig: localConfig,
		syncItems:   syncItems,
		selected:    make(map[int]bool),
	}, nil
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.syncItems.SyncItems)-1 {
				m.cursor++
			}

		case " ":
			// Toggle selection
			if len(m.syncItems.SyncItems) > 0 {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}

		case "a":
			// Select all
			for i := range m.syncItems.SyncItems {
				m.selected[i] = true
			}

		case "n":
			// Select none
			m.selected = make(map[int]bool)

		case "s":
			// Sync selected items
			return m, m.syncSelected()

		case "enter":
			// Show item details (placeholder)
			if len(m.syncItems.SyncItems) > 0 && m.cursor < len(m.syncItems.SyncItems) {
				item := m.syncItems.SyncItems[m.cursor]
				m.lastStatus = fmt.Sprintf("Details for: %s", item.Name)
				m.showStatus = true
			}

		case "r":
			// Refresh data
			return m, m.refreshData()
		}

	case statusMsg:
		m.lastStatus = msg.message
		m.showStatus = true
		if msg.isError {
			m.err = fmt.Errorf(msg.message)
		} else {
			m.err = nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m tuiModel) View() string {
	if m.err != nil {
		return mainBorderStyle.Render(errorStyle.Render(fmt.Sprintf("❌ Error: %v\n\nPress 'q' to quit.", m.err)))
	}

	var b strings.Builder

	// Fancy title with emoji
	title := titleStyle.Render("🚀 Sync Station v1.0.0")
	
	// Header info bar with icons and formatting
	computerInfo := fmt.Sprintf("💻 %s     ☁️  %s     🔄 %d items", 
		m.localConfig.CurrentComputer, 
		m.localConfig.CloudSyncDir,
		len(m.syncItems.SyncItems))
	headerInfo := headerInfoStyle.Render(computerInfo)
	
	b.WriteString(title + "\n")
	b.WriteString(headerInfo + "\n\n")

	// Content box with fancy border
	var contentBuilder strings.Builder
	
	// Items header with fancy styling
	itemsHeader := itemsHeaderStyle.Render("📦 Sync Items")
	contentBuilder.WriteString(itemsHeader + "\n")
	
	if len(m.syncItems.SyncItems) == 0 {
		emptyMsg := dimmedStyle.Render("📭 No sync items configured yet")
		helpMsg := dimmedStyle.Render("💡 Add items with: syncstation add")
		contentBuilder.WriteString(emptyMsg + "\n")
		contentBuilder.WriteString(helpMsg + "\n")
	} else {
		for i, item := range m.syncItems.SyncItems {
			// Fancy cursor indicator
			cursor := "  "
			if m.cursor == i {
				cursor = "▶ "
			}

			// Enhanced checkbox with fancy symbols
			checkbox := "☐"
			if m.selected[i] {
				checkbox = "☑"
			}

			// Get status and apply fancy styling
			status := m.getItemStatus(item)
			styledStatus := m.getStyledStatus(status)
			
			// Get item type icon
			typeIcon := m.getTypeIcon(item.Type)
			
			// Get local path with fancy formatting
			localPath := item.GetCurrentComputerPath(m.localConfig.CurrentComputer)
			pathInfo := "⚠️  No path configured"
			if localPath != "" {
				pathInfo = pathStyle.Render(localPath)
			}

			// File count if available
			fileCount := m.getFileCount(item)
			
			// Main item line with fancy formatting
			itemLine := fmt.Sprintf("%s%s %s %s %s", 
				cursor, checkbox, typeIcon, item.Name, styledStatus)
			
			// Sub-line with path and details
			subLine := fmt.Sprintf("    %s %s", pathInfo, fileCount)
			
			if m.cursor == i {
				contentBuilder.WriteString(selectedItemStyle.Render(itemLine) + "\n")
				contentBuilder.WriteString(selectedItemStyle.Render(subLine) + "\n")
			} else {
				contentBuilder.WriteString(itemStyle.Render(itemLine) + "\n")
				contentBuilder.WriteString(dimmedStyle.Render(subLine) + "\n")
			}
			
			// Add spacing between items
			contentBuilder.WriteString("\n")
		}
	}

	// Wrap content in fancy box
	content := contentBoxStyle.Render(contentBuilder.String())
	b.WriteString(content)

	// Status message with fancy styling
	if m.showStatus && m.lastStatus != "" {
		statusMsg := fmt.Sprintf("ℹ️  %s", m.lastStatus)
		b.WriteString("\n" + warningStyle.Render(statusMsg) + "\n")
	}

	// Fancy help bar
	helpText := "💡 [Space] select  [S] sync  [P] push  [L] pull  [A] all  [N] none  [R] refresh  [Q] quit"
	helpBar := helpBarStyle.Render(helpText)
	b.WriteString("\n" + helpBar)

	// Wrap everything in main border
	return mainBorderStyle.Render(b.String())
}

// getStyledStatus returns a styled status with fancy colors and icons
func (m tuiModel) getStyledStatus(status string) string {
	switch status {
	case "Ready":
		return syncedStyle.Render("🟢 Synced")
	case "Local newer":
		return localNewerStyle.Render("🔵 Local newer")
	case "Cloud newer":
		return cloudNewerStyle.Render("🟣 Cloud newer")
	case "Conflict":
		return conflictStyle.Render("🔴 Conflicts")
	case "Local missing":
		return warningStyle.Render("🟡 Local missing")
	case "Cloud missing":
		return warningStyle.Render("🟡 Cloud missing")
	case "No path":
		return dimmedStyle.Render("⚪ No path")
	default:
		return dimmedStyle.Render("⚪ Unknown")
	}
}

// getTypeIcon returns an emoji icon based on item type
func (m tuiModel) getTypeIcon(itemType string) string {
	switch itemType {
	case "folder":
		return "📁"
	case "file":
		return "📄"
	default:
		return "📦"
	}
}

// getFileCount returns a styled file count for display
func (m tuiModel) getFileCount(item *config.SyncItem) string {
	// For now, show placeholder - would be enhanced with actual file counting
	localPath := item.GetCurrentComputerPath(m.localConfig.CurrentComputer)
	if localPath == "" {
		return dimmedStyle.Render("(not configured)")
	}
	
	if !config.PathExists(config.ExpandPath(localPath)) {
		return dimmedStyle.Render("(missing)")
	}
	
	// TODO: Implement actual file counting
	if item.Type == "folder" {
		return fileCountStyle.Render("(folder)")
	} else {
		return fileCountStyle.Render("(file)")
	}
}

// getItemStatus determines the sync status of an item
func (m tuiModel) getItemStatus(item *config.SyncItem) string {
	localPath := item.GetCurrentComputerPath(m.localConfig.CurrentComputer)
	if localPath == "" {
		return "No path"
	}

	if !config.PathExists(localPath) {
		return "Local missing"
	}

	cloudPath := item.GetCloudPath(m.localConfig.GetCloudConfigsPath())
	if !config.PathExists(cloudPath) {
		return "Cloud missing"
	}

	// TODO: Implement proper status checking in Phase 6
	return "Ready"
}

// getStatusIcon returns a colored icon for the status
func (m tuiModel) getStatusIcon(status string) string {
	switch status {
	case "Ready":
		return syncedStyle.Render("✓")
	case "Local missing", "Cloud missing", "No path":
		return warningStyle.Render("⚠")
	case "Conflict":
		return conflictStyle.Render("✗")
	default:
		return warningStyle.Render("ℹ")
	}
}

// syncSelected creates a command to sync selected items
func (m tuiModel) syncSelected() tea.Cmd {
	return func() tea.Msg {
		selectedCount := 0
		for range m.selected {
			selectedCount++
		}

		if selectedCount == 0 {
			return statusMsg{
				message: "No items selected for sync",
				isError: true,
			}
		}

		// TODO: Implement actual sync in Phase 6
		return statusMsg{
			message: fmt.Sprintf("Sync for %d items will be implemented in Phase 6", selectedCount),
			isError: false,
		}
	}
}

// refreshData creates a command to refresh sync items data
func (m tuiModel) refreshData() tea.Cmd {
	return func() tea.Msg {
		// Reload sync items
		_, err := config.LoadSyncItemsData(m.localConfig.GetSyncItemsPath())
		if err != nil {
			return statusMsg{
				message: fmt.Sprintf("Failed to refresh: %v", err),
				isError: true,
			}
		}

		// TODO: In a proper implementation, we'd send the new data back
		// For now, just confirm the refresh worked
		return statusMsg{
			message: "Data refreshed",
			isError: false,
		}
	}
}

// LaunchTUI launches the interactive terminal user interface
func LaunchTUI() error {
	model, err := InitialTUIModel()
	if err != nil {
		return err
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// getConfigDir returns the appropriate config directory for the platform
func getConfigDir() string {
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