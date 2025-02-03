# skukozh

A command-line tool to find and extract content from files in a directory. The tool is particularly useful for preparing code files for analysis by AI models like Claude or GPT.

## Features

- Find files by extension in a directory and create a file list
- Generate a formatted content file suitable for AI analysis
- Support for multiple file extensions
- Clean output format with file paths, types, and content boundaries
- Preserves original file paths in output

## Building

Make sure you have Go installed on your system, then:

```bash
# Clone the repository
git clone <your-repository-url>
cd skukozh

# Build the binary
go build -o skukozh
```

## Usage

### Finding Files

To find files with specific extensions and create a file list:

```bash
# Find PHP and JavaScript files
./skukozh -ext 'php,js' find /path/to/directory

# Find PHP files only
./skukozh -ext 'php' find /path/to/directory

# Find all files (no extension filter)
./skukozh find /path/to/directory
```

This will create `skukozh_file_list.txt` with relative paths to all matching files.

### Generating Content File

To generate a content file from the file list:

```bash
./skukozh gen /path/to/directory
```

This will create `skukozh_result.txt` containing the content of all files in a format suitable for AI analysis:

```
### FILE: application/index.php
### TYPE: php
### CONTENT START ###
```php
<?php
// File content here
```
### CONTENT END ###
```

## Output Format

The generated content file includes:
- Clear file boundaries
- File paths and types
- Language-specific code blocks
- Content start/end markers

This format is optimized for AI models to easily parse and understand the structure of your codebase.

## Special Thanks

Special thanks to Claude.ai for assistance in developing this tool and optimizing the output format for AI analysis.

## License

[Your chosen license]

## Contributing

Feel free to open issues or submit pull requests!
