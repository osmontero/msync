package sync

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/osmontero/msync/internal/utils"
)

// Options holds configuration for the synchronization process
type Options struct {
	Checksum    bool   // Use checksum comparison
	DryRun      bool   // Show what would be copied without copying
	Interactive bool   // Interactive mode (not used in sync package directly)
	Verbose     bool   // Enable verbose output
	Recursive   bool   // Recursively sync directories
	Delete      bool   // Delete extraneous files from destination
	Threads     int    // Number of concurrent threads
	Method      string // Comparison method: mtime, checksum, size
}

// Syncer represents a file synchronizer
type Syncer struct {
	options Options
	stats   Stats
	mu      sync.Mutex // For thread-safe stats updates
}

// Stats holds synchronization statistics
type Stats struct {
	FilesChecked int64
	FilesCopied  int64
	FilesDeleted int64
	BytesCopied  int64
	BytesDeleted int64
	DirsCreated  int64
	Errors       []string
	// Preview-specific stats
	FilesToCopy   int64
	FilesToDelete int64
	BytesToCopy   int64
	BytesToDelete int64
	DirsToCreate  int64
}

// FileInfo represents file information for comparison
type FileInfo struct {
	Path     string
	Size     int64
	ModTime  time.Time
	Checksum string
	IsDir    bool
}

// New creates a new Syncer with the given options
func New(options Options) *Syncer {
	// Set defaults for critical options
	if options.Method == "" {
		options.Method = "mtime"
	}
	if options.Threads <= 0 {
		options.Threads = 4
	}

	return &Syncer{
		options: options,
		stats:   Stats{},
	}
}

// Sync performs synchronization from source to destination
func (s *Syncer) Sync(source, destination string) error {
	if s.options.Verbose {
		fmt.Printf("Starting sync from %s to %s\n", source, destination)
		fmt.Printf("Method: %s, Threads: %d, DryRun: %t\n",
			s.options.Method, s.options.Threads, s.options.DryRun)
	}

	startTime := time.Now()

	// Build file maps for comparison
	sourceFiles, err := s.buildFileMap(source, "")
	if err != nil {
		return fmt.Errorf("failed to scan source directory: %w", err)
	}

	var destFiles map[string]FileInfo
	if _, err := os.Stat(destination); err == nil {
		destFiles, err = s.buildFileMap(destination, "")
		if err != nil {
			return fmt.Errorf("failed to scan destination directory: %w", err)
		}
	} else {
		destFiles = make(map[string]FileInfo)
		// Create destination directory
		if !s.options.DryRun {
			if err := os.MkdirAll(destination, 0755); err != nil {
				return fmt.Errorf("failed to create destination directory: %w", err)
			}
		}
	}

	// Process files with worker pool
	if err := s.processFiles(source, destination, sourceFiles, destFiles); err != nil {
		return err
	}

	// Handle file deletion if requested
	if s.options.Delete {
		if err := s.deleteExtraFiles(destination, sourceFiles, destFiles); err != nil {
			return err
		}
	}

	elapsed := time.Since(startTime)
	if s.options.Verbose {
		s.printStats(elapsed)
	}

	return nil
}

// buildFileMap creates a map of files in the given directory
func (s *Syncer) buildFileMap(root, relativeRoot string) (map[string]FileInfo, error) {
	files := make(map[string]FileInfo)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.addError(fmt.Sprintf("Error accessing %s: %v", path, err))
			return nil // Continue walking
		}

		// Calculate relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Skip subdirectories if not recursive
		if !s.options.Recursive && info.IsDir() && filepath.Dir(relPath) != "." {
			return filepath.SkipDir
		}

		fileInfo := FileInfo{
			Path:    relPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}

		// Calculate checksum if needed and it's a regular file
		if s.shouldCalculateChecksum() && !info.IsDir() {
			checksum, err := s.calculateChecksum(path)
			if err != nil {
				s.addError(fmt.Sprintf("Failed to calculate checksum for %s: %v", path, err))
			} else {
				fileInfo.Checksum = checksum
			}
		}

		files[relPath] = fileInfo
		s.incrementChecked()

		return nil
	})

	return files, err
}

// processFiles handles the actual file synchronization using worker pools
func (s *Syncer) processFiles(source, dest string, sourceFiles, destFiles map[string]FileInfo) error {
	// Create work queue
	workChan := make(chan FileInfo, len(sourceFiles))
	errorChan := make(chan error, len(sourceFiles))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.options.Threads; i++ {
		wg.Add(1)
		go s.worker(source, dest, workChan, errorChan, &wg)
	}

	// Queue work items
	go func() {
		defer close(workChan)
		for _, sourceFile := range sourceFiles {
			if s.shouldSync(sourceFile, destFiles) {
				workChan <- sourceFile
			}
		}
	}()

	// Wait for completion
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// Collect errors
	for err := range errorChan {
		if err != nil {
			s.addError(err.Error())
		}
	}

	return nil
}

// worker processes files from the work queue
func (s *Syncer) worker(source, dest string, workChan <-chan FileInfo, errorChan chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	for fileInfo := range workChan {
		sourcePath := filepath.Join(source, fileInfo.Path)
		destPath := filepath.Join(dest, fileInfo.Path)

		if err := s.syncFile(sourcePath, destPath, fileInfo); err != nil {
			errorChan <- err
		}
	}
}

// shouldSync determines if a file needs to be synchronized
func (s *Syncer) shouldSync(sourceFile FileInfo, destFiles map[string]FileInfo) bool {
	destFile, exists := destFiles[sourceFile.Path]

	if !exists {
		return true // File doesn't exist in destination
	}

	if sourceFile.IsDir != destFile.IsDir {
		return true // Type mismatch (file vs directory)
	}

	if sourceFile.IsDir {
		return false // Directories don't need content sync
	}

	// Compare based on the selected method
	switch s.options.Method {
	case "size":
		return sourceFile.Size != destFile.Size
	case "checksum":
		return s.compareByChecksum(sourceFile, destFile)
	case "mtime":
		fallthrough
	default:
		return sourceFile.ModTime.After(destFile.ModTime) || sourceFile.Size != destFile.Size
	}
}

// compareByChecksum compares files by their checksums
func (s *Syncer) compareByChecksum(sourceFile, destFile FileInfo) bool {
	if sourceFile.Checksum != "" && destFile.Checksum != "" {
		return sourceFile.Checksum != destFile.Checksum
	}
	// If checksums aren't available, fall back to size + mtime
	return sourceFile.Size != destFile.Size || sourceFile.ModTime.After(destFile.ModTime)
}

// syncFile synchronizes a single file
func (s *Syncer) syncFile(sourcePath, destPath string, fileInfo FileInfo) error {
	if fileInfo.IsDir {
		return s.syncDirectory(destPath, fileInfo)
	}
	return s.syncRegularFile(sourcePath, destPath, fileInfo)
}

// syncDirectory creates a directory
func (s *Syncer) syncDirectory(destPath string, fileInfo FileInfo) error {
	if s.options.Verbose {
		if s.options.DryRun {
			fmt.Printf("Would create directory: %s\n", destPath)
		} else {
			fmt.Printf("Creating directory: %s\n", destPath)
		}
	}

	if s.options.DryRun {
		s.incrementDirToCreate()
	} else {
		if err := os.MkdirAll(destPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destPath, err)
		}
		s.incrementDirCreated()
	}

	return nil
}

// syncRegularFile copies a regular file
func (s *Syncer) syncRegularFile(sourcePath, destPath string, fileInfo FileInfo) error {
	if s.options.Verbose {
		if s.options.DryRun {
			fmt.Printf("Would copy: %s -> %s (%s)\n", sourcePath, destPath, utils.FormatBytes(fileInfo.Size))
		} else {
			fmt.Printf("Copying: %s -> %s (%s)\n", sourcePath, destPath, utils.FormatBytes(fileInfo.Size))
		}
	}

	if s.options.DryRun {
		s.incrementFileToCopy(fileInfo.Size)
		return nil
	}

	// Get source file info for preserving timestamps
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy file
	if err := s.copyFile(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to copy file %s: %w", sourcePath, err)
	}

	// Preserve both access and modification times from source
	if err := os.Chtimes(destPath, sourceInfo.ModTime(), sourceInfo.ModTime()); err != nil {
		s.addError(fmt.Sprintf("Failed to preserve timestamps for %s: %v", destPath, err))
	}

	s.incrementCopied(fileInfo.Size)
	return nil
}

// copyFile performs the actual file copy
func (s *Syncer) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// deleteExtraFiles removes files from destination that don't exist in source
func (s *Syncer) deleteExtraFiles(dest string, sourceFiles, destFiles map[string]FileInfo) error {
	for relPath := range destFiles {
		if _, exists := sourceFiles[relPath]; !exists {
			fullPath := filepath.Join(dest, relPath)

			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}

			if s.options.Verbose {
				sizeStr := ""
				if !info.IsDir() {
					sizeStr = fmt.Sprintf(" (%s)", utils.FormatBytes(info.Size()))
				}
				if s.options.DryRun {
					fmt.Printf("Would delete: %s%s\n", fullPath, sizeStr)
				} else {
					fmt.Printf("Deleting: %s%s\n", fullPath, sizeStr)
				}
			}

			if s.options.DryRun {
				if !info.IsDir() {
					s.incrementFileToDelete(info.Size())
				}
			} else {
				if err := os.RemoveAll(fullPath); err != nil {
					s.addError(fmt.Sprintf("Failed to delete %s: %v", fullPath, err))
					continue
				}

				if !info.IsDir() {
					s.incrementDeleted(info.Size())
				}
			}
		}
	}
	return nil
}

// calculateChecksum calculates SHA256 checksum of a file
func (s *Syncer) calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// shouldCalculateChecksum determines if checksums should be calculated
func (s *Syncer) shouldCalculateChecksum() bool {
	return s.options.Method == "checksum" || s.options.Checksum
}

// Thread-safe statistics methods
func (s *Syncer) incrementChecked() {
	s.mu.Lock()
	s.stats.FilesChecked++
	s.mu.Unlock()
}

func (s *Syncer) incrementCopied(bytes int64) {
	s.mu.Lock()
	s.stats.FilesCopied++
	s.stats.BytesCopied += bytes
	s.mu.Unlock()
}

func (s *Syncer) incrementDeleted(bytes int64) {
	s.mu.Lock()
	s.stats.FilesDeleted++
	s.stats.BytesDeleted += bytes
	s.mu.Unlock()
}

func (s *Syncer) addError(err string) {
	s.mu.Lock()
	s.stats.Errors = append(s.stats.Errors, err)
	s.mu.Unlock()
}

func (s *Syncer) incrementDirCreated() {
	s.mu.Lock()
	s.stats.DirsCreated++
	s.mu.Unlock()
}

func (s *Syncer) incrementDirToCreate() {
	s.mu.Lock()
	s.stats.DirsToCreate++
	s.mu.Unlock()
}

func (s *Syncer) incrementFileToCopy(bytes int64) {
	s.mu.Lock()
	s.stats.FilesToCopy++
	s.stats.BytesToCopy += bytes
	s.mu.Unlock()
}

func (s *Syncer) incrementFileToDelete(bytes int64) {
	s.mu.Lock()
	s.stats.FilesToDelete++
	s.stats.BytesToDelete += bytes
	s.mu.Unlock()
}

// printStats prints synchronization statistics
func (s *Syncer) printStats(elapsed time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.options.DryRun {
		s.printPreviewSummary(elapsed)
	} else {
		s.printExecutionSummary(elapsed)
	}
}

// printPreviewSummary prints a comprehensive preview of planned operations
func (s *Syncer) printPreviewSummary(elapsed time.Duration) {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("                    SYNC PREVIEW SUMMARY\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")

	totalOperations := s.stats.FilesToCopy + s.stats.FilesToDelete + s.stats.DirsToCreate

	if totalOperations == 0 {
		fmt.Printf("✓ No changes needed - source and destination are in sync\n")
		fmt.Printf("  Files checked: %d\n", s.stats.FilesChecked)
		fmt.Printf("  Analysis time: %s\n", utils.FormatDuration(elapsed.Seconds()))
		return
	}

	fmt.Printf("📋 PLANNED OPERATIONS:\n")
	fmt.Printf(strings.Repeat("-", 30) + "\n")

	if s.stats.FilesToCopy > 0 {
		fmt.Printf("📁 Files to copy:      %d (%s)\n", s.stats.FilesToCopy, utils.FormatBytes(s.stats.BytesToCopy))
	}

	if s.stats.DirsToCreate > 0 {
		fmt.Printf("📂 Directories to create: %d\n", s.stats.DirsToCreate)
	}

	if s.stats.FilesToDelete > 0 {
		fmt.Printf("🗑️  Files to delete:    %d (%s)\n", s.stats.FilesToDelete, utils.FormatBytes(s.stats.BytesToDelete))
	}

	fmt.Printf(strings.Repeat("-", 30) + "\n")
	fmt.Printf("📊 SUMMARY:\n")
	fmt.Printf("   Total operations:   %d\n", totalOperations)
	fmt.Printf("   Files checked:      %d\n", s.stats.FilesChecked)
	fmt.Printf("   Net data transfer:  %s\n", utils.FormatBytes(s.stats.BytesToCopy-s.stats.BytesToDelete))
	fmt.Printf("   Analysis time:      %s\n", utils.FormatDuration(elapsed.Seconds()))

	if s.stats.BytesToCopy > 0 {
		// Rough estimation: 50MB/s for typical operations
		estimatedSeconds := float64(s.stats.BytesToCopy) / (50 * 1024 * 1024)
		if estimatedSeconds < 1 {
			estimatedSeconds = 1
		}
		fmt.Printf("   Estimated sync time: %s\n", utils.FormatDuration(estimatedSeconds))
	}

	if len(s.stats.Errors) > 0 {
		fmt.Printf("\n⚠️  ISSUES FOUND (%d):\n", len(s.stats.Errors))
		for _, err := range s.stats.Errors {
			fmt.Printf("   • %s\n", err)
		}
	}

	fmt.Printf(strings.Repeat("=", 60) + "\n")
	fmt.Printf("💡 To execute these changes, run the same command without --dry-run\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")
}

// printExecutionSummary prints statistics for actual sync operations
func (s *Syncer) printExecutionSummary(elapsed time.Duration) {
	fmt.Printf("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("            SYNCHRONIZATION COMPLETE\n")
	fmt.Printf(strings.Repeat("=", 50) + "\n")

	fmt.Printf("📊 RESULTS:\n")
	fmt.Printf("   Files checked:  %d\n", s.stats.FilesChecked)
	fmt.Printf("   Files copied:   %d\n", s.stats.FilesCopied)
	fmt.Printf("   Files deleted:  %d\n", s.stats.FilesDeleted)
	fmt.Printf("   Dirs created:   %d\n", s.stats.DirsCreated)
	fmt.Printf("   Bytes copied:   %s\n", utils.FormatBytes(s.stats.BytesCopied))
	fmt.Printf("   Bytes deleted:  %s\n", utils.FormatBytes(s.stats.BytesDeleted))
	fmt.Printf("   Time elapsed:   %s\n", utils.FormatDuration(elapsed.Seconds()))

	if s.stats.BytesCopied > 0 && elapsed.Seconds() > 0 {
		throughput := float64(s.stats.BytesCopied) / elapsed.Seconds()
		fmt.Printf("   Throughput:     %s/s\n", utils.FormatBytes(int64(throughput)))
	}

	if len(s.stats.Errors) > 0 {
		fmt.Printf("\n⚠️  ERRORS (%d):\n", len(s.stats.Errors))
		for _, err := range s.stats.Errors {
			fmt.Printf("   • %s\n", err)
		}
	} else {
		fmt.Printf("\n✅ Synchronization completed successfully!\n")
	}

	fmt.Printf(strings.Repeat("=", 50) + "\n")
}
