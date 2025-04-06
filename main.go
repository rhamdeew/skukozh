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
	"text/tabwriter"
	"unicode"
)

var (
	extFlag   = flag.String("ext", "", "Comma-separated list of file extensions (e.g., 'php,js,ts')")
	shortExt  = flag.String("e", "", "Short form of -ext flag")
	countFlag = flag.Int("count", 20, "Number of largest files to show in analyze command")
	noIgnore  = flag.Bool("no-ignore", false, "Don't apply default ignore patterns")
	verbose   = flag.Bool("verbose", false, "Show verbose output while finding files")

	// Variable for os.Exit that can be overridden in tests
	osExit = os.Exit
)

// Common directories to ignore
var ignoredDirs = []string{
	"node_modules",
	"vendor",
	"dist",
	"build",
	".git",
	".svn",
	".hg",
	"bower_components",
	"target",
	"bin",
	"obj",
}

// Common binary/non-text file extensions
var binaryFileExts = []string{
	// Images
	".jpg", ".jpeg", ".png", ".gif", ".bmp", ".ico", ".svg", ".webp",
	// Audio
	".mp3", ".wav", ".ogg", ".flac", ".aac", ".m4a",
	// Video
	".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv", ".webm",
	// Archives
	".zip", ".tar", ".gz", ".rar", ".7z", ".jar", ".war",
	// Binaries
	".exe", ".dll", ".so", ".dylib", ".bin", ".dat",
	// Other binary formats
	".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
}

// Common text file extensions that are always allowed
var commonTextExts = []string{
	// Programming languages
	".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".h", ".hpp", ".cs", ".php", ".rb", ".rs", ".swift",
	// Web
	".html", ".htm", ".css", ".scss", ".sass", ".less", ".jsx", ".tsx", ".vue", ".svelte",
	// Config files
	".json", ".yaml", ".yml", ".toml", ".xml", ".ini", ".env",
	// Documentation
	".md", ".txt", ".rst", ".adoc",
	// Shell scripts
	".sh", ".bash", ".zsh", ".fish", ".bat", ".cmd", ".ps1",
}

const usage = `Usage:
  skukozh [-e|-ext 'ext1,ext2,...'] [-no-ignore] [-verbose] find|f <directory>  - Find files and create file list
  skukozh gen|g <directory>                                                      - Generate content file from file list
  skukozh [-count N] analyze|a                                                   - Analyze the result file (default top 20 files)
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
		return // This ensures the function stops here in tests
	}

	if len(files) == 0 {
		fmt.Println("No files found! Use -no-ignore flag to include hidden files and directories.")
		return
	}

	// Write to file
	output := strings.Join(files, "\n")
	err = os.WriteFile("skukozh_file_list.txt", []byte(output), 0644)
	if err != nil {
		fmt.Printf("Error writing file list: %v\n", err)
		osExit(1)
		return // This ensures the function stops here in tests
	}

	fmt.Printf("Found %d files. File list saved to skukozh_file_list.txt\n", len(files))
}

// findFilesInternal is a testable version of findFiles that returns errors instead of exiting
func findFilesInternal(root string, supportedExts []string) ([]string, error) {
	var files []string
	debugMode := *verbose || os.Getenv("SKUKOZH_DEBUG") == "1"

	if len(supportedExts) == 0 {
		// If no extensions are specified, use common text extensions
		supportedExts = commonTextExts
	}

	// Make sure the root is an absolute path
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if the root path exists and is a directory
	rootInfo, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot access directory: %w", err)
	}
	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", absRoot)
	}

	if debugMode {
		fmt.Printf("Scanning directory: %s\n", absRoot)
	}

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if debugMode {
				fmt.Printf("Error accessing path %s: %v\n", path, err)
			}
			return nil // Skip errors and continue
		}

		// Get relative path for proper display in messages
		relPath, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)

		// Skip root directory itself
		if path == absRoot {
			return nil
		}

		// Check if it's a hidden file or directory
		if !*noIgnore && isHidden(d.Name()) {
			if d.IsDir() {
				if debugMode {
					fmt.Printf("Skipping hidden directory: %s\n", relPath)
				}
				return filepath.SkipDir
			}
			if debugMode {
				fmt.Printf("Skipping hidden file: %s\n", relPath)
			}
			return nil
		}

		// Skip go build files
		if d.IsDir() && strings.HasPrefix(d.Name(), "_") {
			if debugMode {
				fmt.Printf("Skipping Go build dir: %s\n", relPath)
			}
			return filepath.SkipDir
		}

		// Skip ignored directories
		if !*noIgnore && d.IsDir() && containsIgnoreCase(ignoredDirs, d.Name()) {
			if debugMode {
				fmt.Printf("Skipping package directory: %s\n", relPath)
			}
			return filepath.SkipDir
		}

		if !d.IsDir() {
			ext := filepath.Ext(path)

			// Skip binary files
			if !*noIgnore && contains(binaryFileExts, strings.ToLower(ext)) {
				if debugMode {
					fmt.Printf("Skipping binary file: %s\n", relPath)
				}
				return nil
			}

			// Skip empty files
			info, err := d.Info()
			if err == nil && info.Size() == 0 {
				if debugMode {
					fmt.Printf("Skipping empty file: %s\n", relPath)
				}
				return nil
			}

			// Check if the file extension matches
			if *noIgnore || len(supportedExts) == 0 || contains(supportedExts, strings.ToLower(ext)) {
				if debugMode {
					fmt.Printf("Adding file: %s\n", relPath)
				}
				files = append(files, relPath)
			} else if debugMode {
				fmt.Printf("Skipping file with unsupported extension: %s\n", relPath)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files for consistent output
	sort.Strings(files)

	if debugMode {
		fmt.Printf("Found %d files\n", len(files))
	}

	return files, nil
}

// isHidden checks if a file or directory is hidden (starts with .)
func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

// containsIgnoreCase checks if a slice contains a string, ignoring case
func containsIgnoreCase(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
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

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(&buf, "\nAnalysis Report")
	fmt.Fprintln(&buf, "==============")
	fmt.Fprintf(&buf, "Total file size: %.2f MB\n", fileSize)
	fmt.Fprintf(&buf, "Total symbols: %d\n\n", symbols)

	if len(files) == 0 {
		fmt.Fprintln(&buf, "No files found in the result file.")
		return buf.String(), nil
	}

	fmt.Fprintf(&buf, "Top %d largest files:\n", topCount)

	// Print table header using tabwriter
	fmt.Fprintln(w, "File\tSize (KB)\tSymbols")
	fmt.Fprintln(w, "────\t────────\t───────")

	// Print file information
	for i, file := range files {
		if i >= topCount {
			break
		}
		fmt.Fprintf(w, "%s\t%.2f\t%d\n",
			file.path,
			float64(file.size)/1024,
			file.symbols)
	}

	w.Flush()
	fmt.Fprintln(&buf, "")

	return buf.String(), nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
