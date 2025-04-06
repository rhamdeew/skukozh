package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

var (
	extFlag   = flag.String("ext", "", "Comma-separated list of file extensions (e.g., 'php,js,ts')")
	shortExt  = flag.String("e", "", "Short form of -ext flag")
	countFlag = flag.Int("count", 20, "Number of largest files to show in analyze command")

	// Variable for os.Exit that can be overridden in tests
	osExit = os.Exit
)

const usage = `Usage:
  skukozh [-e|-ext 'ext1,ext2,...'] find|f <directory>  - Find files and create file list
  skukozh gen|g <directory>                             - Generate content file from file list
  skukozh [-count N] analyze|a                          - Analyze the result file (default top 20 files)
`

type FileInfo struct {
	path    string
	size    int64
	symbols int
}

func main() {
	// Call run, which contains the actual logic
	os.Exit(run())
}

// run handles the command execution and returns the exit code
func run() int {
	// Support both -ext and --ext
	flag.CommandLine.Init("skukozh", flag.ContinueOnError)
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Print(usage)
		return 1
	}

	// Parse supported extensions from either -ext or -e flag
	var supportedExts []string
	extValue := *extFlag
	if *shortExt != "" {
		extValue = *shortExt
	}
	if extValue != "" {
		exts := strings.Split(extValue, ",")
		for _, ext := range exts {
			ext = strings.TrimSpace(ext)
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			supportedExts = append(supportedExts, ext)
		}
	}

	command := args[0]
	switch command {
	case "find", "f":
		if len(args) != 2 {
			fmt.Print(usage)
			return 1
		}
		directory := args[1]
		findFiles(directory, supportedExts)

	case "gen", "g":
		if len(args) != 2 {
			fmt.Print(usage)
			return 1
		}
		directory := args[1]
		generateContentFile(directory)

	case "analyze", "a":
		if len(args) != 1 {
			fmt.Print(usage)
			return 1
		}
		analyzeResultFile(*countFlag)

	default:
		fmt.Print(usage)
		return 1
	}

	return 0
}

func findFiles(root string, supportedExts []string) {
	files, err := findFilesInternal(root, supportedExts)
	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		osExit(1)
	}

	// Write to file
	output := strings.Join(files, "\n")
	err = os.WriteFile("skukozh_file_list.txt", []byte(output), 0644)
	if err != nil {
		fmt.Printf("Error writing file list: %v\n", err)
		osExit(1)
	}

	fmt.Println("File list saved to skukozh_file_list.txt")
}

// findFilesInternal is a testable version of findFiles that returns errors instead of exiting
func findFilesInternal(root string, supportedExts []string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			ext := filepath.Ext(path)
			if len(supportedExts) == 0 || contains(supportedExts, ext) {
				// Convert to relative path and use forward slashes
				relPath, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				relPath = filepath.ToSlash(relPath)
				files = append(files, relPath)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files for consistent output
	sort.Strings(files)

	return files, nil
}

func generateContentFile(baseDir string) {
	result, err := generateContentFileInternal(baseDir)
	if err != nil {
		fmt.Printf("Error reading file list: %v\n", err)
		osExit(1)
	}

	// Write result file
	err = os.WriteFile("skukozh_result.txt", []byte(result), 0644)
	if err != nil {
		fmt.Printf("Error writing result file: %v\n", err)
		osExit(1)
	}

	fmt.Println("Content file saved to skukozh_result.txt")
}

// generateContentFileInternal is a testable version that returns errors instead of exiting
func generateContentFileInternal(baseDir string) (string, error) {
	// Read file list
	content, err := os.ReadFile("skukozh_file_list.txt")
	if err != nil {
		return "", err
	}

	files := strings.Split(string(content), "\n")
	var output strings.Builder

	for _, file := range files {
		if file == "" {
			continue
		}

		// Combine base directory with file path for reading
		fullPath := filepath.Join(baseDir, file)

		// Read file content
		fileContent, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", fullPath, err)
			continue
		}

		// Remove blank lines
		lines := strings.Split(string(fileContent), "\n")
		var nonEmptyLines []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				nonEmptyLines = append(nonEmptyLines, line)
			}
		}
		fileContent = []byte(strings.Join(nonEmptyLines, "\n"))

		// Write file section with original path
		ext := filepath.Ext(file)
		output.WriteString(fmt.Sprintf("#FILE %s\n", file))
		output.WriteString(fmt.Sprintf("#TYPE %s\n", strings.TrimPrefix(ext, ".")))
		output.WriteString("#START\n")
		output.WriteString("```" + strings.TrimPrefix(ext, ".") + "\n")
		output.Write(fileContent)
		if !bytes.HasSuffix(fileContent, []byte("\n")) {
			output.WriteString("\n")
		}
		output.WriteString("```\n")
		output.WriteString("#END\n\n")
	}

	return output.String(), nil
}

func analyzeResultFile(topCount int) {
	output, err := analyzeResultFileInternal(topCount)
	if err != nil {
		fmt.Printf("Error reading result file: %v\n", err)
		osExit(1)
	}

	fmt.Print(output)
}

// analyzeResultFileInternal is a testable version that returns errors instead of exiting
func analyzeResultFileInternal(topCount int) (string, error) {
	content, err := os.ReadFile("skukozh_result.txt")
	if err != nil {
		return "", err
	}

	// Calculate total file size
	fileSize := float64(len(content)) / (1024 * 1024) // Convert to MB

	// Count total symbols (excluding whitespace)
	symbols := 0
	for _, r := range string(content) {
		if !unicode.IsSpace(r) {
			symbols++
		}
	}

	// Parse file sections and collect information
	sections := strings.Split(string(content), "#FILE ")
	var files []FileInfo

	for _, section := range sections[1:] { // Skip first empty section
		lines := strings.Split(section, "\n")
		if len(lines) < 1 {
			continue
		}

		filePath := strings.TrimSpace(lines[0])

		// Find content between START and END markers
		startMarker := "#START\n```"
		endMarker := "```\n#END"

		startIdx := strings.Index(section, startMarker)
		if startIdx == -1 {
			continue
		}
		startIdx += len(startMarker)

		// Find the language identifier line
		nextNewline := strings.Index(section[startIdx:], "\n")
		if nextNewline == -1 {
			continue
		}
		startIdx += nextNewline + 1

		endIdx := strings.Index(section[startIdx:], endMarker)
		if endIdx == -1 {
			continue
		}

		fileContent := section[startIdx : startIdx+endIdx]
		symbolCount := 0
		for _, r := range fileContent {
			if !unicode.IsSpace(r) {
				symbolCount++
			}
		}

		files = append(files, FileInfo{
			path:    filePath,
			size:    int64(len(fileContent)),
			symbols: symbolCount,
		})
	}

	// Sort files by size
	sort.Slice(files, func(i, j int) bool {
		return files[i].size > files[j].size
	})

	// Find the longest file path for formatting
	maxPathLen := 0
	for _, file := range files {
		if len(file.path) > maxPathLen {
			maxPathLen = len(file.path)
		}
	}
	// Ensure minimum width and add padding
	if maxPathLen < 50 {
		maxPathLen = 50
	}
	maxPathLen += 2 // Add some padding

	var result strings.Builder

	// Print header
	result.WriteString("\nAnalysis Report\n")
	result.WriteString("==============\n\n")
	result.WriteString(fmt.Sprintf("Total file size: %.2f MB\n", fileSize))
	result.WriteString(fmt.Sprintf("Total symbols: %d\n\n", symbols))

	if len(files) == 0 {
		result.WriteString("No files found in the result file.\n")
		return result.String(), nil
	}

	result.WriteString(fmt.Sprintf("Top %d largest files:\n", topCount))

	// Print table header with proper spacing
	headerFormat := fmt.Sprintf("%%-%ds %%12s %%15s\n", maxPathLen)
	result.WriteString(fmt.Sprintf(headerFormat, "File", "Size (KB)", "Symbols"))

	// Print separator with proper length
	result.WriteString(fmt.Sprintf("%s %s %s\n",
		strings.Repeat("─", maxPathLen),
		strings.Repeat("─", 12),
		strings.Repeat("─", 15)))

	// Print file information
	fileFormat := fmt.Sprintf("%%-%ds %%12.2f %%15d\n", maxPathLen)
	for i, file := range files {
		if i >= topCount {
			break
		}
		result.WriteString(fmt.Sprintf(fileFormat,
			file.path,
			float64(file.size)/1024,
			file.symbols))
	}
	result.WriteString("\n")

	return result.String(), nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
