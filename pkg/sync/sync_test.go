package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	opts := Options{
		Verbose:   true,
		Recursive: true,
		Threads:   8,
	}

	syncer := New(opts)

	if syncer.options.Verbose != true {
		t.Error("Expected Verbose to be true")
	}

	if syncer.options.Threads != 8 {
		t.Error("Expected Threads to be 8")
	}
}

func TestCalculateChecksum(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	syncer := New(Options{})
	checksum, err := syncer.calculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	// Expected SHA256 of "Hello, World!"
	expected := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
	if checksum != expected {
		t.Errorf("Expected checksum %s, got %s", expected, checksum)
	}
}

func TestShouldSync(t *testing.T) {
	syncer := New(Options{Method: "mtime"})

	now := time.Now()
	older := now.Add(-time.Hour)

	sourceFile := FileInfo{
		Path:    "test.txt",
		Size:    100,
		ModTime: now,
		IsDir:   false,
	}

	// Test file doesn't exist in destination
	destFiles := make(map[string]FileInfo)
	if !syncer.shouldSync(sourceFile, destFiles) {
		t.Error("Should sync when file doesn't exist in destination")
	}

	// Test file exists but is older
	destFiles["test.txt"] = FileInfo{
		Path:    "test.txt",
		Size:    100,
		ModTime: older,
		IsDir:   false,
	}

	if !syncer.shouldSync(sourceFile, destFiles) {
		t.Error("Should sync when destination file is older")
	}

	// Test file exists and is newer
	destFiles["test.txt"] = FileInfo{
		Path:    "test.txt",
		Size:    100,
		ModTime: now.Add(time.Hour),
		IsDir:   false,
	}

	if syncer.shouldSync(sourceFile, destFiles) {
		t.Error("Should not sync when destination file is newer")
	}
}

func TestSyncDirectoryStructure(t *testing.T) {
	// Create temporary directories
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	// Create source structure
	if err := os.MkdirAll(filepath.Join(sourceDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testContent := "test content"
	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	nestedFile := filepath.Join(sourceDir, "subdir", "nested.txt")
	if err := os.WriteFile(nestedFile, []byte("nested content"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	// Test sync
	syncer := New(Options{Recursive: true, Verbose: true})
	err := syncer.Sync(sourceDir, destDir)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	t.Logf("Source dir: %s", sourceDir)
	t.Logf("Dest dir: %s", destDir)

	// Verify files were copied
	destTestFile := filepath.Join(destDir, "test.txt")
	if _, err := os.Stat(destTestFile); os.IsNotExist(err) {
		t.Error("Test file was not copied to destination")
	}

	destNestedFile := filepath.Join(destDir, "subdir", "nested.txt")
	if _, err := os.Stat(destNestedFile); os.IsNotExist(err) {
		t.Error("Nested file was not copied to destination")
	}

	// Verify content
	content, err := os.ReadFile(destTestFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("File content mismatch. Expected %s, got %s", testContent, string(content))
	}
}

func TestDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	destDir := filepath.Join(tmpDir, "dest")

	// Create source file
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test dry run
	syncer := New(Options{DryRun: true})
	err := syncer.Sync(sourceDir, destDir)
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}

	// Verify file was NOT copied
	destTestFile := filepath.Join(destDir, "test.txt")
	if _, err := os.Stat(destTestFile); !os.IsNotExist(err) {
		t.Error("File should not exist after dry run")
	}
}
