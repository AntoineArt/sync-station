package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiffLine represents a line in a diff
type DiffLine struct {
	LineNumber int
	Content    string
	Type       string // "same", "added", "removed", "modified"
}

// FileDiff represents the difference between two files
type FileDiff struct {
	LocalPath     string
	CloudPath     string
	LocalExists   bool
	CloudExists   bool
	LocalModTime  time.Time
	CloudModTime  time.Time
	Status        string // "same", "local_newer", "cloud_newer", "conflict", "local_only", "cloud_only"
	Lines         []DiffLine
}

// DiffEngine handles file comparison and diff generation
type DiffEngine struct {
	config *Config
}

// NewDiffEngine creates a new diff engine
func NewDiffEngine(config *Config) *DiffEngine {
	return &DiffEngine{
		config: config,
	}
}

// CompareFiles compares two files and returns their diff
func (d *DiffEngine) CompareFiles(localPath, cloudPath string) (*FileDiff, error) {
	diff := &FileDiff{
		LocalPath: localPath,
		CloudPath: cloudPath,
	}

	// Check if files exist
	localInfo, localErr := os.Stat(localPath)
	cloudInfo, cloudErr := os.Stat(cloudPath)

	diff.LocalExists = localErr == nil
	diff.CloudExists = cloudErr == nil

	if diff.LocalExists {
		diff.LocalModTime = localInfo.ModTime()
	}
	if diff.CloudExists {
		diff.CloudModTime = cloudInfo.ModTime()
	}

	// Determine status
	if !diff.LocalExists && !diff.CloudExists {
		diff.Status = "neither_exist"
		return diff, nil
	}
	if !diff.LocalExists {
		diff.Status = "cloud_only"
		return diff, nil
	}
	if !diff.CloudExists {
		diff.Status = "local_only"
		return diff, nil
	}

	// Both files exist, compare content and timestamps
	contentSame, err := d.compareFileContent(localPath, cloudPath)
	if err != nil {
		return nil, err
	}

	if contentSame {
		diff.Status = "same"
	} else {
		// Files are different, determine which is newer
		if diff.LocalModTime.After(diff.CloudModTime) {
			diff.Status = "local_newer"
		} else if diff.CloudModTime.After(diff.LocalModTime) {
			diff.Status = "cloud_newer"
		} else {
			diff.Status = "conflict" // Same timestamp but different content
		}
	}

	// Generate line-by-line diff for text files
	if d.isTextFile(localPath) && d.isTextFile(cloudPath) {
		diff.Lines, err = d.generateLineDiff(localPath, cloudPath)
		if err != nil {
			return nil, err
		}
	}

	return diff, nil
}

// compareFileContent compares the content of two files
func (d *DiffEngine) compareFileContent(path1, path2 string) (bool, error) {
	content1, err := os.ReadFile(path1)
	if err != nil {
		return false, err
	}

	content2, err := os.ReadFile(path2)
	if err != nil {
		return false, err
	}

	return string(content1) == string(content2), nil
}

// generateLineDiff generates a line-by-line diff between two files
func (d *DiffEngine) generateLineDiff(path1, path2 string) ([]DiffLine, error) {
	lines1, err := d.readFileLines(path1)
	if err != nil {
		return nil, err
	}

	lines2, err := d.readFileLines(path2)
	if err != nil {
		return nil, err
	}

	return d.computeDiff(lines1, lines2), nil
}

// readFileLines reads a file and returns its lines
func (d *DiffEngine) readFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// computeDiff computes the diff between two sets of lines
// This is a simple implementation - in a real application you'd want to use a proper diff algorithm
func (d *DiffEngine) computeDiff(lines1, lines2 []string) []DiffLine {
	var diff []DiffLine
	
	maxLen := len(lines1)
	if len(lines2) > maxLen {
		maxLen = len(lines2)
	}

	for i := 0; i < maxLen; i++ {
		line1 := ""
		line2 := ""
		
		if i < len(lines1) {
			line1 = lines1[i]
		}
		if i < len(lines2) {
			line2 = lines2[i]
		}

		if line1 == line2 {
			diff = append(diff, DiffLine{
				LineNumber: i + 1,
				Content:    line1,
				Type:       "same",
			})
		} else if line1 == "" {
			diff = append(diff, DiffLine{
				LineNumber: i + 1,
				Content:    line2,
				Type:       "added",
			})
		} else if line2 == "" {
			diff = append(diff, DiffLine{
				LineNumber: i + 1,
				Content:    line1,
				Type:       "removed",
			})
		} else {
			// Both lines exist but are different
			diff = append(diff, DiffLine{
				LineNumber: i + 1,
				Content:    "- " + line1,
				Type:       "removed",
			})
			diff = append(diff, DiffLine{
				LineNumber: i + 1,
				Content:    "+ " + line2,
				Type:       "added",
			})
		}
	}

	return diff
}

// isTextFile checks if a file is likely a text file
func (d *DiffEngine) isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExtensions := []string{".txt", ".json", ".yaml", ".yml", ".md", ".conf", ".config", ".ini", ".log"}
	
	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}
	
	return false
}

// GetSyncItemDiff returns the diff for all files in a sync item
func (d *DiffEngine) GetSyncItemDiff(syncItem *SyncItem) (map[string]*FileDiff, error) {
	diffs := make(map[string]*FileDiff)
	
	localPath := d.config.GetCurrentComputerPath(syncItem)
	cloudPath := d.config.GetCloudPath(syncItem)
	
	if localPath == "" {
		return diffs, fmt.Errorf("no path configured for current computer")
	}
	
	// Get all files in both locations
	localFiles, err := d.getFilesInDirectory(localPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	cloudFiles, err := d.getFilesInDirectory(cloudPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	// Create a set of all files
	allFiles := make(map[string]bool)
	for _, file := range localFiles {
		allFiles[file] = true
	}
	for _, file := range cloudFiles {
		allFiles[file] = true
	}
	
	// Compare each file
	for file := range allFiles {
		localFilePath := filepath.Join(localPath, file)
		cloudFilePath := filepath.Join(cloudPath, file)
		
		diff, err := d.CompareFiles(localFilePath, cloudFilePath)
		if err != nil {
			return nil, err
		}
		
		diffs[file] = diff
	}
	
	return diffs, nil
}

// getFilesInDirectory returns all files in a directory (recursively)
func (d *DiffEngine) getFilesInDirectory(dirPath string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}
		
		return nil
	})
	
	return files, err
}