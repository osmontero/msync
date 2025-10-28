package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkCalculateChecksum(b *testing.B) {
	// Create a temporary file for benchmarking
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create a 1MB file
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	syncer := New(Options{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := syncer.calculateChecksum(testFile)
		if err != nil {
			b.Fatalf("Failed to calculate checksum: %v", err)
		}
	}
}

func BenchmarkBuildFileMap(b *testing.B) {
	// Create a directory structure with many files
	tmpDir := b.TempDir()

	// Create 100 files
	for i := 0; i < 100; i++ {
		content := "test content for benchmarking"
		testFile := filepath.Join(tmpDir, fmt.Sprintf("test%d.txt", i))
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	syncer := New(Options{Recursive: true})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := syncer.buildFileMap(tmpDir, "")
		if err != nil {
			b.Fatalf("Failed to build file map: %v", err)
		}
	}
}

func BenchmarkSync(b *testing.B) {
	// Create source directory with files
	tmpDir := b.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		b.Fatalf("Failed to create source directory: %v", err)
	}

	// Create 50 files
	for i := 0; i < 50; i++ {
		content := "benchmark test content for file"
		testFile := filepath.Join(sourceDir, fmt.Sprintf("test%d.txt", i))
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	syncer := New(Options{Recursive: true, Threads: 4})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		destDir := filepath.Join(tmpDir, fmt.Sprintf("dest%d", i))
		err := syncer.Sync(sourceDir, destDir)
		if err != nil {
			b.Fatalf("Failed to sync: %v", err)
		}
	}
}
