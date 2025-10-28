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

### From Source
```bash
git clone https://github.com/osmontero/msync.git
cd msync
make build
sudo make install
```

### Using Go
```bash
go install github.com/osmontero/msync/cmd@latest
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

#### Dry Run Preview
```bash
# Preview what would be synchronized
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
  -v, --verbose           Enable verbose output
  -r, --recursive         Sync directories recursively (default: true)
      --delete            Delete files in destination not present in source
  -j, --threads N         Number of concurrent threads (default: 4)
      --method METHOD     Comparison method: mtime, checksum, size (default: mtime)
  -h, --help              Show help message
      --version           Show version information
```

### Comparison Methods

| Method | Description | Speed | Accuracy | Use Case |
|--------|-------------|--------|----------|----------|
| `mtime` | Modification time + size | Fast | Good | General sync, frequent updates |
| `checksum` | SHA256 hash | Slow | Excellent | Critical data, verification |
| `size` | File size only | Very Fast | Basic | Large files, quick checks |

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

# Preview deployment changes
msync --dry-run --delete ./build/ /var/www/html/
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

## Development

### Building from Source
```bash
# Clone repository
git clone https://github.com/osmontero/msync.git
cd msync

# Install dependencies
make deps

# Build binary
make build

# Run tests
make test

# Run benchmarks
make bench

# Build for multiple platforms
make build-all
```

### Testing
```bash
# Run unit tests
go test ./pkg/sync

# Run benchmarks
go test -bench=. ./pkg/sync

# Test coverage
make test-coverage
```

### Project Structure
```
msync/
├── cmd/                    # Main application
├── pkg/sync/              # Core synchronization library
├── internal/utils/        # Utility functions
├── Makefile              # Build automation
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

## Roadmap

- [ ] Network synchronization support (SSH)
- [ ] Configuration file support
- [ ] Compression during transfer
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