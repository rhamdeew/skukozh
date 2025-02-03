package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	extFlag = flag.String("ext", "", "Comma-separated list of file extensions (e.g., 'php,js,ts')")
)

const usage = `Usage: 
  skukozh [-ext 'ext1,ext2,...'] find <directory>  - Find files and create file list
  skukozh gen <directory>                          - Generate content file from file list
`

func main() {
	// Support both -ext and --ext
	flag.CommandLine.Init("skukozh", flag.ContinueOnError)
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Print(usage)
		os.Exit(1)
	}

	// Parse supported extensions
	var supportedExts []string
	if *extFlag != "" {
		exts := strings.Split(*extFlag, ",")
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
	case "find":
		if len(args) != 2 {
			fmt.Print(usage)
			os.Exit(1)
		}
		directory := args[1]
		findFiles(directory, supportedExts)
	
	case "gen":
		if len(args) != 2 {
			fmt.Print(usage)
			os.Exit(1)
		}
		directory := args[1]
		generateContentFile(directory)
	
	default:
		fmt.Print(usage)
		os.Exit(1)
	}
}

func findFiles(root string, supportedExts []string) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
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
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}

	// Sort files for consistent output
	sort.Strings(files)

	// Write to file
	output := strings.Join(files, "\n")
	err = ioutil.WriteFile("skukozh_file_list.txt", []byte(output), 0644)
	if err != nil {
		fmt.Printf("Error writing file list: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("File list saved to skukozh_file_list.txt")
}

func generateContentFile(baseDir string) {
	// Read file list
	content, err := ioutil.ReadFile("skukozh_file_list.txt")
	if err != nil {
		fmt.Printf("Error reading file list: %v\n", err)
		os.Exit(1)
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
		fileContent, err := ioutil.ReadFile(fullPath)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", fullPath, err)
			continue
		}

		// Write file section with original path
		ext := filepath.Ext(file)
		output.WriteString(fmt.Sprintf("### FILE: %s\n", file))
		output.WriteString(fmt.Sprintf("### TYPE: %s\n", strings.TrimPrefix(ext, ".")))
		output.WriteString("### CONTENT START ###\n")
		output.WriteString("```" + strings.TrimPrefix(ext, ".") + "\n")
		output.Write(fileContent)
		if !bytes.HasSuffix(fileContent, []byte("\n")) {
			output.WriteString("\n")
		}
		output.WriteString("```\n")
		output.WriteString("### CONTENT END ###\n\n")
	}

	// Write result file
	err = ioutil.WriteFile("skukozh_result.txt", []byte(output.String()), 0644)
	if err != nil {
		fmt.Printf("Error writing result file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Content file saved to skukozh_result.txt")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
