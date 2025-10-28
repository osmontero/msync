package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/osmontero/msync/pkg/sync"
)

var (
	version   = "1.0.0"
	buildTime = "unknown"
)

type Config struct {
	Source      string
	Destination string
	Checksum    bool
	DryRun      bool
	Interactive bool
	Verbose     bool
	Recursive   bool
	Delete      bool
	Threads     int
	Method      string
	ShowHelp    bool
	ShowVersion bool
}

func main() {
	config := parseFlags()

	if config.ShowHelp {
		printUsage()
		return
	}

	if config.ShowVersion {
		fmt.Printf("msync version %s\n", version)
		fmt.Printf("Build time: %s\n", buildTime)
		return
	}

	if config.Source == "" || config.Destination == "" {
		fmt.Fprintf(os.Stderr, "Error: Both source and destination paths are required\n\n")
		printUsage()
		os.Exit(1)
	}

	// Validate paths
	if _, err := os.Stat(config.Source); os.IsNotExist(err) {
		log.Fatalf("Source path does not exist: %s", config.Source)
	}

	// Create destination directory if it doesn't exist
	if _, err := os.Stat(config.Destination); os.IsNotExist(err) {
		if err := os.MkdirAll(config.Destination, 0755); err != nil {
			log.Fatalf("Failed to create destination directory: %v", err)
		}
	}

	// Create synchronizer
	syncOptions := sync.Options{
		Checksum:    config.Checksum,
		DryRun:      config.DryRun,
		Interactive: config.Interactive,
		Verbose:     config.Verbose,
		Recursive:   config.Recursive,
		Delete:      config.Delete,
		Threads:     config.Threads,
		Method:      config.Method,
	}

	syncer := sync.New(syncOptions)

	// Handle interactive mode
	if config.Interactive {
		// First run a dry run to show preview
		previewOptions := syncOptions
		previewOptions.DryRun = true
		previewOptions.Verbose = true
		previewSyncer := sync.New(previewOptions)
		
		fmt.Println("🔍 Analyzing changes...")
		if err := previewSyncer.Sync(config.Source, config.Destination); err != nil {
			log.Fatalf("Preview analysis failed: %v", err)
		}

		// Ask for confirmation
		if !askForConfirmation() {
			fmt.Println("Operation cancelled by user.")
			return
		}

		// Proceed with actual sync
		actualOptions := syncOptions
		actualOptions.DryRun = false
		syncer = sync.New(actualOptions)
	}

	// Perform synchronization
	if err := syncer.Sync(config.Source, config.Destination); err != nil {
		log.Fatalf("Synchronization failed: %v", err)
	}

	if config.Verbose {
		fmt.Println("Synchronization completed successfully")
	}
}

func parseFlags() Config {
	config := Config{
		Threads: 4,       // Default number of threads
		Method:  "mtime", // Default comparison method
	}

	flag.StringVar(&config.Source, "source", "", "Source path to sync from")
	flag.StringVar(&config.Source, "s", "", "Source path to sync from (short)")
	flag.StringVar(&config.Destination, "dest", "", "Destination path to sync to")
	flag.StringVar(&config.Destination, "d", "", "Destination path to sync to (short)")
	flag.BoolVar(&config.Checksum, "checksum", false, "Use checksum comparison instead of modification time")
	flag.BoolVar(&config.Checksum, "c", false, "Use checksum comparison (short)")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Show what would be copied without actually copying")
	flag.BoolVar(&config.DryRun, "plan", false, "Preview changes without executing (alias for --dry-run)")
	flag.BoolVar(&config.DryRun, "n", false, "Dry run (short)")
	flag.BoolVar(&config.Interactive, "interactive", false, "Show preview and ask for confirmation before proceeding")
	flag.BoolVar(&config.Interactive, "i", false, "Interactive mode (short)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose output (short)")
	flag.BoolVar(&config.Recursive, "recursive", true, "Recursively sync directories")
	flag.BoolVar(&config.Recursive, "r", true, "Recursive (short)")
	flag.BoolVar(&config.Delete, "delete", false, "Delete extraneous files from destination")
	flag.IntVar(&config.Threads, "threads", 4, "Number of concurrent threads")
	flag.IntVar(&config.Threads, "j", 4, "Number of threads (short)")
	flag.StringVar(&config.Method, "method", "mtime", "Comparison method: mtime, checksum, size")
	flag.BoolVar(&config.ShowHelp, "help", false, "Show help")
	flag.BoolVar(&config.ShowHelp, "h", false, "Show help (short)")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version")

	flag.Parse()

	// Handle positional arguments if flags not used
	args := flag.Args()
	if len(args) >= 1 && config.Source == "" {
		config.Source = args[0]
	}
	if len(args) >= 2 && config.Destination == "" {
		config.Destination = args[1]
	}

	return config
}

func printUsage() {
	fmt.Printf(`msync v%s - Fast file synchronization tool

Usage:
  msync [OPTIONS] SOURCE DEST
  msync -s SOURCE -d DEST [OPTIONS]

Examples:
  msync /home/user/docs /backup/docs
  msync -c -v /src /dst                    # Use checksum with verbose output
  msync --plan --delete /src /dst          # Preview sync with deletion
  msync -i /src /dst                       # Interactive mode with preview
  msync -j 8 --method checksum /src /dst   # Use 8 threads with checksum

Options:
  -s, --source PATH       Source directory or file
  -d, --dest PATH         Destination directory or file
  -c, --checksum          Use checksum comparison (slower but more accurate)
  -n, --dry-run           Show what would be synced without making changes
      --plan              Preview changes without executing (same as --dry-run)
  -i, --interactive       Show preview and ask for confirmation before proceeding
  -v, --verbose           Enable verbose output
  -r, --recursive         Sync directories recursively (default: true)
      --delete            Delete files in destination not present in source
  -j, --threads N         Number of concurrent threads (default: 4)
      --method METHOD     Comparison method: mtime, checksum, size (default: mtime)
  -h, --help              Show this help message
      --version           Show version information

Comparison Methods:
  mtime    - Compare by modification time (fastest)
  checksum - Compare by SHA256 hash (most accurate)
  size     - Compare by file size (fast but less reliable)

`, version)
}

// askForConfirmation prompts the user for confirmation
func askForConfirmation() bool {
	fmt.Print("\n❓ Do you want to proceed with these changes? [y/N]: ")
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
