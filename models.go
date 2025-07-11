package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Computer represents a computer in the sync configuration
type Computer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SyncItem represents a configuration item that can be synced
type SyncItem struct {
	Name    string            `json:"name"`
	Paths   map[string]string `json:"paths"`   // computerID -> path
	Enabled bool              `json:"enabled"` // whether this item is enabled for sync
}

// FileStatus represents the status of a file in sync
type FileStatus struct {
	Path         string    `json:"path"`
	LocalExists  bool      `json:"localExists"`
	CloudExists  bool      `json:"cloudExists"`
	LocalModTime time.Time `json:"localModTime"`
	CloudModTime time.Time `json:"cloudModTime"`
	Status       string    `json:"status"` // "synced", "local_newer", "cloud_newer", "conflict", "missing"
}

// Config represents the main configuration structure
type Config struct {
	CurrentComputer string                `json:"currentComputer"`
	Computers       map[string]*Computer  `json:"computers"`
	SyncItems       []*SyncItem           `json:"syncItems"`
	ConfigsPath     string                `json:"configsPath"` // path to configs folder
	LastSync        time.Time             `json:"lastSync"`
}

// NewConfig creates a new empty configuration
func NewConfig() *Config {
	return &Config{
		Computers:   make(map[string]*Computer),
		SyncItems:   make([]*SyncItem, 0),
		ConfigsPath: "configs",
		LastSync:    time.Now(),
	}
}

// LoadConfig loads configuration from file
func LoadConfig(filename string) (*Config, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return NewConfig(), nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func (c *Config) SaveConfig(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// AddComputer adds a new computer to the configuration
func (c *Config) AddComputer(id, name string) {
	c.Computers[id] = &Computer{
		ID:   id,
		Name: name,
	}
}

// AddSyncItem adds a new sync item to the configuration
func (c *Config) AddSyncItem(name string, paths map[string]string) {
	syncItem := &SyncItem{
		Name:    name,
		Paths:   paths,
		Enabled: true,
	}
	c.SyncItems = append(c.SyncItems, syncItem)
}

// GetCurrentComputerPath returns the path for the current computer for a given sync item
func (c *Config) GetCurrentComputerPath(syncItem *SyncItem) string {
	if path, exists := syncItem.Paths[c.CurrentComputer]; exists {
		return path
	}
	return ""
}

// GetCloudPath returns the cloud storage path for a sync item
func (c *Config) GetCloudPath(syncItem *SyncItem) string {
	return filepath.Join(c.ConfigsPath, syncItem.Name)
}

// DetectCurrentComputer tries to detect the current computer based on hostname
func (c *Config) DetectCurrentComputer() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}

	// Check if this hostname matches any computer ID
	for id := range c.Computers {
		if id == hostname {
			c.CurrentComputer = id
			return id
		}
	}

	// If not found, add this computer
	c.AddComputer(hostname, hostname)
	c.CurrentComputer = hostname
	return hostname
}