package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
)

// Available slash commands
var slashCommands = []string{
	"/help",
	"/exit",
	"/clear",
	"/config",
	"/tools",
	"/prompt",
	"/version",
}

// PathCompleter implements readline.AutoCompleter for file path completion
// and slash command completion
type PathCompleter struct{}

// Do provides path completion and slash command completion functionality
func (p *PathCompleter) Do(line []rune, pos int) (newLine [][]rune, offset int) {
	lineStr := string(line[:pos])
	
	// Handle slash commands at the beginning of the line
	if strings.HasPrefix(lineStr, "/") && !strings.Contains(lineStr[:pos], " ") {
		var candidates [][]rune
		for _, cmd := range slashCommands {
			if strings.HasPrefix(cmd, lineStr) {
				candidates = append(candidates, []rune(cmd))
			}
		}
		if len(candidates) > 0 {
			return candidates, 0
		}
		// If no slash command matches, fall through to path completion
	}
	
	// If not at the beginning or not a slash command, try path completion
	
	// Find the start of the path
	pathStart := strings.LastIndexAny(lineStr, " \t")
	if pathStart == -1 {
		pathStart = 0
	} else {
		pathStart++
	}
	
	path := lineStr[pathStart:]
	_ = lineStr[:pathStart] // Prefix part of the line (unused after fix)
	
	// Handle relative paths and home directory
	var dirPath, filePrefix string
	if path == "" || path == "." {
		dirPath = "."
		filePrefix = ""
	} else if strings.HasPrefix(path, "~/") || path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, 0
		}
		if path == "~" {
			dirPath = homeDir
			filePrefix = ""
		} else {
			dirPath = filepath.Join(homeDir, path[2:])
			if strings.HasSuffix(path, "/") {
				filePrefix = ""
			} else {
				dirPath = filepath.Dir(dirPath)
				filePrefix = filepath.Base(path)
			}
		}
	} else {
		if strings.HasSuffix(path, "/") {
			dirPath = path
			filePrefix = ""
		} else {
			dirPath = filepath.Dir(path)
			filePrefix = filepath.Base(path)
		}
	}
	
	// Read directory entries
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, 0
	}
	defer dir.Close()
	
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, 0
	}
	
	var candidates []string
	for _, name := range names {
		// Skip hidden files unless prefix starts with a dot
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(filePrefix, ".") {
			continue
		}
		
		if strings.HasPrefix(name, filePrefix) {
			fullPath := filepath.Join(dirPath, name)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}
			
			// Add trailing slash for directories
			if info.IsDir() {
				name += "/"
			}
			
			candidates = append(candidates, name)
		}
	}
	
	// Build completion candidates
	var suggestions [][]rune
	for _, candidate := range candidates {
		// Only use the candidate name, not the full path with prefix
		suggestions = append(suggestions, []rune(candidate))
	}
	
	// Return the offset as pathStart to ensure the completion only replaces
	// the path portion, not the entire line
	return suggestions, pathStart
}

// GetPathCompleter returns a new PathCompleter instance
func GetPathCompleter() readline.AutoCompleter {
	return &PathCompleter{}
}