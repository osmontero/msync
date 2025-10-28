package tar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTarArchive_Create(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "msync-tar-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, World!"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	subDir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	subFile := filepath.Join(subDir, "sub.txt")
	if err := os.WriteFile(subFile, []byte("Subdirectory file"), 0644); err != nil {
		t.Fatalf("Failed to create sub file: %v", err)
	}

	// Test creating TAR archive
	archivePath := filepath.Join(tempDir, "test.tar")
	options := TarOptions{
		Compression: false,
		GPGEncrypt:  false,
		GPGSign:     false,
		Verbose:     true,
	}

	archive, err := New(archivePath, options)
	if err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	if err := archive.Create(sourceDir); err != nil {
		t.Fatalf("Failed to create TAR archive: %v", err)
	}

	// Verify archive was created
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Fatalf("Archive file was not created")
	}
}

func TestTarArchive_Extract(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "msync-tar-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	testFile := filepath.Join(sourceDir, "test.txt")
	testContent := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create TAR archive
	archivePath := filepath.Join(tempDir, "test.tar")
	options := TarOptions{
		Compression: false,
		GPGEncrypt:  false,
		GPGSign:     false,
		Verbose:     true,
	}

	archive, err := New(archivePath, options)
	if err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	if err := archive.Create(sourceDir); err != nil {
		t.Fatalf("Failed to create TAR archive: %v", err)
	}

	// Extract to different directory
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("Failed to create extract dir: %v", err)
	}

	if err := archive.Extract(extractDir); err != nil {
		t.Fatalf("Failed to extract TAR archive: %v", err)
	}

	// Verify extracted file
	extractedFile := filepath.Join(extractDir, "test.txt")
	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(content) != testContent {
		t.Fatalf("Extracted content doesn't match: got %s, want %s", string(content), testContent)
	}
}

func TestTarArchive_List(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "msync-tar-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, World!"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	subDir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create TAR archive
	archivePath := filepath.Join(tempDir, "test.tar")
	options := TarOptions{
		Compression: false,
		GPGEncrypt:  false,
		GPGSign:     false,
		Verbose:     false,
	}

	archive, err := New(archivePath, options)
	if err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	if err := archive.Create(sourceDir); err != nil {
		t.Fatalf("Failed to create TAR archive: %v", err)
	}

	// List archive contents
	files, err := archive.List()
	if err != nil {
		t.Fatalf("Failed to list archive contents: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files in archive, got %d", len(files))
	}

	// Check if expected files are present
	fileNames := make(map[string]bool)
	for _, file := range files {
		fileNames[file.Name] = true
	}

	if !fileNames["test.txt"] {
		t.Fatalf("test.txt not found in archive listing")
	}

	if !fileNames["subdir"] {
		t.Fatalf("subdir not found in archive listing")
	}
}

func TestTarArchive_Compression(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "msync-tar-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	sourceDir := filepath.Join(tempDir, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create a larger test file to see compression benefits
	testFile := filepath.Join(sourceDir, "test.txt")
	largeContent := make([]byte, 10000)
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}
	if err := os.WriteFile(testFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create compressed TAR archive
	archivePath := filepath.Join(tempDir, "test.tar.gz")
	options := TarOptions{
		Compression: true,
		GPGEncrypt:  false,
		GPGSign:     false,
		Verbose:     true,
	}

	archive, err := New(archivePath, options)
	if err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	if err := archive.Create(sourceDir); err != nil {
		t.Fatalf("Failed to create compressed TAR archive: %v", err)
	}

	// Extract and verify
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("Failed to create extract dir: %v", err)
	}

	if err := archive.Extract(extractDir); err != nil {
		t.Fatalf("Failed to extract compressed TAR archive: %v", err)
	}

	// Verify extracted content
	extractedFile := filepath.Join(extractDir, "test.txt")
	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if len(content) != len(largeContent) {
		t.Fatalf("Extracted content size doesn't match: got %d, want %d", len(content), len(largeContent))
	}
}

func TestIsTarFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.tar", true},
		{"test.tar.gz", true},
		{"test.tgz", true},
		{"test.tar.gpg", true},
		{"test.tar.gz.gpg", true},
		{"test.tgz.gpg", true},
		{"test.txt", false},
		{"test.zip", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsTarFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsTarFile(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestParseTarOptions(t *testing.T) {
	tests := []struct {
		path             string
		expectCompression bool
		expectEncryption  bool
	}{
		{"test.tar", false, false},
		{"test.tar.gz", true, false},
		{"test.tgz", true, false},
		{"test.tar.gpg", false, true},
		{"test.tar.gz.gpg", true, true},
		{"test.tgz.gpg", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			options := ParseTarOptions(tt.path)
			if options.Compression != tt.expectCompression {
				t.Errorf("ParseTarOptions(%s).Compression = %v, want %v", tt.path, options.Compression, tt.expectCompression)
			}
			if options.GPGEncrypt != tt.expectEncryption {
				t.Errorf("ParseTarOptions(%s).GPGEncrypt = %v, want %v", tt.path, options.GPGEncrypt, tt.expectEncryption)
			}
		})
	}
}