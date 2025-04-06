# Skukozh

A command-line tool to find and extract content from files in a directory. The tool is particularly useful for preparing code files for analysis by AI models like Claude or GPT.

![CI/CD Status](https://github.com/rhamdeew/skukozh/actions/workflows/build.yml/badge.svg)

## Features

- Find files by extension in a directory and create a file list
- Generate a formatted content file suitable for AI analysis
- Support for multiple file extensions
- Clean output format with file paths, types, and content boundaries
- Preserves original file paths in output
- Removes blank lines to optimize token usage
- Supports both long and short command formats

## Usage

### Finding Files

To find files with specific extensions and create a file list:

```bash
# Find PHP and JavaScript files (long format)
./skukozh --ext 'php,js' find /path/to/directory

# Find PHP and JavaScript files (short format)
./skukozh -e 'php,js' f /path/to/directory

# Find PHP files only
./skukozh -e 'php' f /path/to/directory

# Find all files (no extension filter)
./skukozh f /path/to/directory
```

This will create `skukozh_file_list.txt` with relative paths to all matching files.

### Generating Content File

To generate a content file from the file list:

```bash
# Long format
./skukozh gen /path/to/directory

# Short format
./skukozh g /path/to/directory
```

This will create `skukozh_result.txt` containing the content of all files in a format suitable for AI analysis, with blank lines removed to optimize token usage:

```
#FILE application/index.php
#TYPE php
#START
"""php
<?php
// File content here
""""
#END
```

### Analyzing Result File

To analyze the generated content file:

```bash
# Show default analysis (top 20 largest files) - long format
./skukozh analyze

# Show default analysis - short format
./skukozh a

# Show top 50 largest files
./skukozh -count 50 analyze
# or
./skukozh -count 50 a
```

This will show:
- Total file size in megabytes
- Total symbol count (excluding whitespace)
- List of largest files with their sizes and symbol counts

Example output:

```
Analysis Report
==============

Total file size: 2.45 MB
Total symbols: 458932

Top 20 largest files:
File                                                Size (KB)        Symbols
--------------------------------------------------     ------        -------
application/models/LargeModel.php                        125.4         24560
application/controllers/MainController.php                98.2         18340
...
```

## Running Tests

To run all tests:
```
go test -v ./...
```

To run specific tests:
```
go test -v -run TestContains
go test -v -run TestFindFiles
go test -v -run TestGenerateContentFile
go test -v -run TestAnalyzeResultFile
go test -v -run TestCLI
```

To see test coverage:
```
go test -v -cover ./...
```

To generate a test coverage report:
```
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Building

Make sure you have Go installed on your system, then:

```bash
# Clone the repository
git clone <your-repository-url>
cd skukozh

# Build the binary
go build -o skukozh
```

## Output Format

The generated content file includes:
- Clear file boundaries
- File paths and types
- Language-specific code blocks
- Content start/end markers
- No blank lines (for token efficiency)

This format is optimized for AI models to easily parse and understand the structure of your codebase while minimizing token usage.

## Command Reference

Long Format | Short Format | Description
-----------|--------------|-------------
`find` | `f` | Find files in directory
`gen` | `g` | Generate content file
`analyze` | `a` | Analyze result file
`--ext` | `-e` | Specify file extensions

## Special Thanks

Special thanks to Claude.ai for assistance in developing this tool and optimizing the output format for AI analysis.

## License

MIT

## Contributing

Feel free to open issues or submit pull requests!
