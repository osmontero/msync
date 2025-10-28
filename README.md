# msync - Fast File Synchronization Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/osmontero/msync)](https://goreportcard.com/report/github.com/osmontero/msync)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

A high-performance, cross-platform file synchronization tool similar to rsync, built in Go. `msync` provides fast, reliable file synchronization with multiple comparison methods, concurrent processing, and comprehensive options for backup and restore operations.

## Features

### Core Capabilities
- **Multiple Comparison Methods**: Choose between modification time, checksum (SHA256), or file size
- **High Performance**: Multi-threaded processing with configurable worker pools
- **Cross-Platform**: Runs on Linux, macOS, Windows, and other Unix-like systems
- **Preserve Attributes**: Maintains file modification times and permissions
- **Directory Synchronization**: Full recursive directory tree synchronization
- **TAR Archive Support**: Create, extract, and synchronize TAR archives with optional compression
- **GPG Integration**: Encrypt and sign TAR archives with GPG for secure backups
- **Dry Run Mode**: Preview operations before execution
- **Delete Support**: Remove extraneous files from destination
- **Progress Reporting**: Detailed statistics and throughput information
- **Verbose Output**: Comprehensive logging of operations

### Performance Features
- **Concurrent Processing**: Configurable number of worker threads
- **Efficient Checksumming**: SHA256 hashing for content verification
- **Memory Efficient**: Optimized file operations and streaming
- **Fast File Walking**: Efficient directory tree traversal

## Installation

### Prerequisites
- **Go 1.19+** for building from source
- **Task** (optional, for development): Install with `brew install go-task/tap/go-task` or `nix-env -iA nixpkgs.go-task`
- **GPG** (optional, for encrypted archives): Usually pre-installed on Unix systems

### From Source
```bash
git clone https://github.com/osmontero/msync.git
cd msync
task build
task install
```

### Using Go
```bash
# Install with proper binary name
go install github.com/osmontero/msync/cmd@latest
# The binary will be installed as 'cmd', rename it:
mv $GOPATH/bin/cmd $GOPATH/bin/msync
# Or if using Go modules (Go 1.16+):
mv $(go env GOPATH)/bin/cmd $(go env GOPATH)/bin/msync

# Alternative: Build locally and install
git clone https://github.com/osmontero/msync.git
cd msync && task build && task install
```

## Usage

### Basic Synchronization
```bash
# Sync source directory to destination
msync /path/to/source /path/to/destination

# Using flags
msync --source /path/to/source --dest /path/to/destination
```

### Advanced Examples

#### Checksum-based Synchronization
```bash
# Use SHA256 checksums for comparison (more accurate)
msync --checksum /source /dest
msync --method checksum /source /dest
```

#### High-Performance Sync
```bash
# Use 16 worker threads for large datasets
msync --threads 16 /source /dest
```

#### Backup with Deletion
```bash
# Mirror source to destination, removing extra files
msync --delete /source /dest
```

#### Preview Mode (Plane Mode)
```bash
# Preview what would be synchronized (comprehensive summary)
msync --plan --verbose /source /dest

# Interactive mode with preview and confirmation
msync --interactive /source /dest

# Traditional dry run (still supported)
msync --dry-run --verbose /source /dest
```

#### Verbose Monitoring
```bash
# Show detailed progress and statistics
msync --verbose /source /dest
```

### Command Line Options

```
Usage:
  msync [OPTIONS] SOURCE DEST
  msync -s SOURCE -d DEST [OPTIONS]

Options:
  -s, --source PATH       Source directory or file
  -d, --dest PATH         Destination directory or file
  -c, --checksum          Use checksum comparison (slower but more accurate)
  -n, --dry-run           Show what would be synced without making changes
      --plan              Preview changes without executing (enhanced dry-run)
  -i, --interactive       Show preview and ask for confirmation before proceeding
  -v, --verbose           Enable verbose output
  -r, --recursive         Sync directories recursively (default: true)
      --delete            Delete files in destination not present in source
  -j, --threads N         Number of concurrent threads (default: 4)
      --method METHOD     Comparison method: mtime, checksum, size (default: mtime)
      --skip-broken-links Skip broken symbolic links entirely

TAR Archive Support:
      --tar-compress      Use gzip compression for TAR files
      --gpg-encrypt       Encrypt TAR files with GPG
      --gpg-sign          Sign TAR files with GPG  
      --gpg-key ID        GPG key ID for encryption/signing
      --gpg-keyring PATH  Path to GPG keyring

General:
  -h, --help              Show help message
      --version           Show version information
```

### Comparison Methods

| Method | Description | Speed | Accuracy | Use Case |
|--------|-------------|--------|----------|----------|
| `mtime` | Modification time + size | Fast | Good | General sync, frequent updates |
| `checksum` | SHA256 hash | Slow | Excellent | Critical data, verification |
| `size` | File size only | Very Fast | Basic | Large files, quick checks |

### TAR Archive Workflows

`msync` automatically detects TAR files and handles three types of operations:

| Source | Destination | Operation | Description |
|--------|-------------|-----------|-------------|
| Directory | `*.tar*` | **Create** | Create TAR archive from directory |
| `*.tar*` | Directory | **Extract** | Extract TAR archive to directory |
| `*.tar*` | `*.tar*` | **Sync** | Extract source, sync, create destination |

**Supported Extensions**: `.tar`, `.tar.gz`, `.tgz`, `.tar.gpg`, `.tar.gz.gpg`, `.tgz.gpg`

## Examples

### Regular Backup
```bash
# Daily backup with progress monitoring
msync -v /home/user/documents /backup/documents

# Mirror backup (exact copy)
msync --delete -v /home/user/projects /backup/projects
```

### Content Verification
```bash
# Verify backup integrity using checksums
msync --method checksum --dry-run /source /backup
```

### High-Performance Scenarios
```bash
# Large dataset synchronization
msync -j 16 -v /data/large_dataset /backup/large_dataset

# Network storage synchronization
msync --threads 8 --method size /local/data /network/storage/data
```

### Development Workflows
```bash
# Sync build output to deployment directory
msync --verbose ./build/ /var/www/html/

# Preview deployment changes with enhanced summary
msync --plan --delete ./build/ /var/www/html/

# Interactive deployment with confirmation
msync --interactive --delete ./build/ /var/www/html/
```

### TAR Archive Operations
```bash
# Create TAR archive from directory
msync /src backup.tar

# Create compressed TAR archive
msync --tar-compress /src backup.tar.gz

# Extract TAR archive to directory
msync archive.tar /dst

# Create encrypted TAR archive
msync --gpg-encrypt --gpg-key USER_ID /src backup.tar.gpg

# Create signed and compressed TAR archive
msync --gpg-sign --tar-compress --gpg-key USER_ID /src backup.tar.gz

# TAR to TAR synchronization
msync old.tar.gz new.tar.gz

# Secure backup workflow
msync --gpg-encrypt --gpg-sign --tar-compress --gpg-key USER_ID /important/data secure_backup.tar.gz.gpg
```

## Enhanced Preview Features

### Comprehensive Preview Summary
The `--plan` flag provides an enhanced dry-run experience with:
- **Visual Summary**: Clear breakdown of planned operations
- **Data Transfer Estimates**: Shows net data transfer and estimated time
- **Operation Categorization**: Separate counts for files to copy, delete, and directories to create
- **Smart Analysis**: Detects when no changes are needed

```bash
# Example output:
============================================================
                    SYNC PREVIEW SUMMARY
============================================================
📋 PLANNED OPERATIONS:
------------------------------
📁 Files to copy:      15 (2.3 MB)
📂 Directories to create: 3
🗑️  Files to delete:    2 (150 KB)
------------------------------
📊 SUMMARY:
   Total operations:   20
   Files checked:      156
   Net data transfer:  2.2 MB
   Analysis time:      0.2s
   Estimated sync time: 1.5s
============================================================
```

### Interactive Mode
The `--interactive` flag combines preview with user confirmation:
1. Performs comprehensive analysis
2. Shows detailed preview summary
3. Prompts for user confirmation
4. Executes only if approved

```bash
msync --interactive /source /dest
# Shows preview, then asks: "Do you want to proceed? [y/N]"
```

## Troubleshooting

### Common Issues

#### Broken Symbolic Links
If you see errors like "Failed to calculate checksum for ... no such file or directory", these are typically broken symbolic links (common in Python virtual environments):

```bash
# Skip broken symlinks entirely
msync --skip-broken-links /source /dest

# Or use verbose mode to see warnings but continue processing
msync --verbose /source /dest
```

#### Python Virtual Environments
Python venv/virtualenv directories contain symlinks that may be broken when moved between systems:

```bash
# Skip broken symlinks when syncing Python projects
msync --skip-broken-links /python-project /backup/python-project
```

#### Permission Errors
Ensure you have read permissions on source files and write permissions on destination:

```bash
# Check permissions
ls -la /source
ls -la /destination

# Run with verbose mode to see detailed error information
msync --verbose /source /dest
```

## Performance

### Benchmarks
On a modern system with SSD storage:
- **Checksum calculation**: ~745µs per MB
- **File map building**: ~300µs for 100 files
- **Synchronization**: ~1.2ms for 50 files

### Optimization Tips
1. **Use appropriate thread count**: Set `--threads` based on your storage and CPU
2. **Choose the right method**: Use `mtime` for speed, `checksum` for accuracy
3. **Batch operations**: Sync larger datasets in single operations
4. **Monitor throughput**: Use `--verbose` to track performance
5. **Handle broken symlinks**: Use `--skip-broken-links` to skip broken symbolic links (common in Python venv directories)

## Development

### Building from Source
```bash
# Clone repository
git clone https://github.com/osmontero/msync.git
cd msync

# Install dependencies
task deps

# Build binary
task build

# Run tests
task test

# Run benchmarks
task bench

# Build for multiple platforms
task build-all

# TAR-specific tests
task test-tar

# Run TAR demos
task demo-tar-basic
task demo-tar-compressed
```

### Testing
```bash
# Run all tests
task test

# Run unit tests for specific packages
go test ./pkg/sync
go test ./pkg/tar

# Run benchmarks
task bench

# Test coverage
task test-coverage

# TAR-specific testing
task test-tar

# GPG functionality demos (requires GPG setup)
task gpg-generate-key
task demo-tar-encrypted
task demo-tar-signed
```

### Project Structure
```
msync/
├── cmd/                    # Main application
├── pkg/
│   ├── sync/              # Core synchronization library
│   └── tar/               # TAR archive handling with GPG support
├── internal/utils/        # Utility functions
├── Taskfile.yaml         # Task automation (replaces Makefile)
└── README.md             # This file
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines
- Write tests for new features
- Follow Go coding conventions
- Update documentation for API changes
- Run benchmarks for performance-critical changes

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Comparison with rsync

| Feature | msync | rsync |
|---------|-------|-------|
| **Language** | Go | C |
| **Performance** | High (multi-threaded) | High (single-threaded) |
| **Cross-platform** | Native binaries | Requires compilation |
| **Checksums** | SHA256 | MD5/SHA1 |
| **Configuration** | Command-line flags | Many options |
| **Network sync** | Local only | SSH/network support |
| **Compression** | No | Yes |
| **Incremental** | Yes | Yes |

## Recent Updates

### ✅ TAR Archive Support (v1.1.0)
- **TAR Operations**: Create, extract, and synchronize TAR archives
- **Compression Support**: Gzip compression (`.tar.gz`, `.tgz`)
- **GPG Integration**: Encrypt and sign TAR archives for secure backups
- **Automatic Detection**: Smart detection of TAR files by extension
- **Flexible Workflows**: Directory ↔ TAR, TAR ↔ TAR synchronization

### Supported TAR Formats
- `.tar` - Basic TAR archive
- `.tar.gz` / `.tgz` - Gzip compressed TAR
- `.tar.gpg` - GPG encrypted TAR
- `.tar.gz.gpg` / `.tgz.gpg` - Compressed and encrypted TAR

## Roadmap

- [x] TAR archive support with compression
- [x] GPG encryption and signing
- [ ] Network synchronization support (SSH)
- [ ] Configuration file support
- [ ] Bandwidth limiting
- [ ] Real-time monitoring API
- [ ] GUI interface
- [ ] Plugin system

## Support

- **Issues**: [GitHub Issues](https://github.com/osmontero/msync/issues)
- **Discussions**: [GitHub Discussions](https://github.com/osmontero/msync/discussions)
- **Documentation**: [Wiki](https://github.com/osmontero/msync/wiki)

---

**msync** - Making file synchronization fast, reliable, and simple.