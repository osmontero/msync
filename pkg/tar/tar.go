package tar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TarOptions holds configuration for TAR operations
type TarOptions struct {
	Compression bool   // Use gzip compression
	GPGEncrypt  bool   // Encrypt the TAR file with GPG
	GPGSign     bool   // Sign the TAR file with GPG
	GPGKeyID    string // GPG key ID for encryption/signing
	GPGKeyring  string // Path to GPG keyring
	Verbose     bool   // Verbose output
}

// TarArchive represents a TAR archive with optional encryption and signing
type TarArchive struct {
	Path    string
	Options TarOptions
	gpg     *GPGHandler
}

// FileInfo represents a file in the TAR archive
type TarFileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
	Mode    os.FileMode
}

// New creates a new TarArchive instance
func New(path string, options TarOptions) (*TarArchive, error) {
	ta := &TarArchive{
		Path:    path,
		Options: options,
	}

	// Initialize GPG handler if encryption or signing is enabled
	if options.GPGEncrypt || options.GPGSign {
		gpgHandler, err := NewGPGHandler(options.GPGKeyring, options.GPGKeyID)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GPG handler: %w", err)
		}
		ta.gpg = gpgHandler
	}

	return ta, nil
}

// Create creates a TAR archive from the specified source directory
func (ta *TarArchive) Create(sourceDir string) error {
	if ta.Options.Verbose {
		fmt.Printf("Creating TAR archive: %s from %s\n", ta.Path, sourceDir)
	}

	// Create the archive file
	file, err := os.Create(ta.Path)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer file.Close()

	var writer io.Writer = file

	// Add gzip compression if enabled
	var gzipWriter *gzip.Writer
	if ta.Options.Compression {
		gzipWriter = gzip.NewWriter(writer)
		writer = gzipWriter
		defer gzipWriter.Close()
	}

	// Add GPG encryption if enabled
	var encryptedWriter io.WriteCloser
	if ta.Options.GPGEncrypt && ta.gpg != nil {
		encryptedWriter, err = ta.gpg.Encrypt(writer)
		if err != nil {
			return fmt.Errorf("failed to create encrypted writer: %w", err)
		}
		writer = encryptedWriter
		defer encryptedWriter.Close()
	}

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Walk through source directory and add files to archive
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path within the archive
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create header for %s: %w", path, err)
		}

		// Set the name to use forward slashes (TAR standard)
		header.Name = filepath.ToSlash(relPath)

		if ta.Options.Verbose {
			fmt.Printf("Adding: %s\n", header.Name)
		}

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write header for %s: %w", path, err)
		}

		// Write file content if it's a regular file
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return fmt.Errorf("failed to write file content for %s: %w", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	// Create GPG signature if enabled
	if ta.Options.GPGSign && ta.gpg != nil {
		signaturePath := ta.Path + ".sig"
		if err := ta.gpg.Sign(ta.Path, signaturePath); err != nil {
			return fmt.Errorf("failed to create GPG signature: %w", err)
		}
		if ta.Options.Verbose {
			fmt.Printf("Created GPG signature: %s\n", signaturePath)
		}
	}

	return nil
}

// Extract extracts the TAR archive to the specified destination directory
func (ta *TarArchive) Extract(destDir string) error {
	if ta.Options.Verbose {
		fmt.Printf("Extracting TAR archive: %s to %s\n", ta.Path, destDir)
	}

	// Verify GPG signature if signing was enabled
	if ta.Options.GPGSign && ta.gpg != nil {
		signaturePath := ta.Path + ".sig"
		if _, err := os.Stat(signaturePath); err == nil {
			if err := ta.gpg.Verify(ta.Path, signaturePath); err != nil {
				return fmt.Errorf("GPG signature verification failed: %w", err)
			}
			if ta.Options.Verbose {
				fmt.Printf("GPG signature verified successfully\n")
			}
		}
	}

	// Open the archive file
	file, err := os.Open(ta.Path)
	if err != nil {
		return fmt.Errorf("failed to open archive file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// Handle GPG decryption if encrypted
	if ta.Options.GPGEncrypt && ta.gpg != nil {
		decryptedReader, err := ta.gpg.Decrypt(reader)
		if err != nil {
			return fmt.Errorf("failed to decrypt archive: %w", err)
		}
		reader = decryptedReader
	}

	// Handle gzip decompression if compressed
	if ta.Options.Compression {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	tarReader := tar.NewReader(reader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Convert back to OS-specific path
		targetPath := filepath.Join(destDir, filepath.FromSlash(header.Name))

		if ta.Options.Verbose {
			fmt.Printf("Extracting: %s\n", header.Name)
		}

		// Handle different file types
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, header.FileInfo().Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			// Create regular file
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				return fmt.Errorf("failed to extract file %s: %w", targetPath, err)
			}

			// Preserve timestamps
			if err := os.Chtimes(targetPath, header.AccessTime, header.ModTime); err != nil {
				fmt.Printf("Warning: failed to set timestamps for %s: %v\n", targetPath, err)
			}

		default:
			fmt.Printf("Warning: unsupported file type %c for %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}

// List returns a list of files in the TAR archive
func (ta *TarArchive) List() ([]TarFileInfo, error) {
	if ta.Options.Verbose {
		fmt.Printf("Listing contents of TAR archive: %s\n", ta.Path)
	}

	// Open the archive file
	file, err := os.Open(ta.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// Handle GPG decryption if encrypted
	if ta.Options.GPGEncrypt && ta.gpg != nil {
		decryptedReader, err := ta.gpg.Decrypt(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt archive: %w", err)
		}
		reader = decryptedReader
	}

	// Handle gzip decompression if compressed
	if ta.Options.Compression {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	tarReader := tar.NewReader(reader)
	var files []TarFileInfo

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		fileInfo := TarFileInfo{
			Name:    header.Name,
			Size:    header.Size,
			ModTime: header.ModTime,
			IsDir:   header.Typeflag == tar.TypeDir,
			Mode:    header.FileInfo().Mode(),
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// IsEncrypted checks if the TAR archive appears to be encrypted
func (ta *TarArchive) IsEncrypted() (bool, error) {
	file, err := os.Open(ta.Path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read first few bytes to check for GPG signature
	header := make([]byte, 10)
	_, err = file.Read(header)
	if err != nil {
		return false, err
	}

	// Check for GPG binary signature (starts with specific bytes)
	return ta.gpg != nil && ta.gpg.IsEncrypted(header), nil
}

// GetFileExtension returns the appropriate file extension for the archive
func (ta *TarArchive) GetFileExtension() string {
	ext := ".tar"
	if ta.Options.Compression {
		ext += ".gz"
	}
	if ta.Options.GPGEncrypt {
		ext += ".gpg"
	}
	return ext
}

// IsTarFile checks if a file path appears to be a TAR archive
func IsTarFile(path string) bool {
	path = strings.ToLower(path)
	return strings.HasSuffix(path, ".tar") ||
		strings.HasSuffix(path, ".tar.gz") ||
		strings.HasSuffix(path, ".tgz") ||
		strings.HasSuffix(path, ".tar.gpg") ||
		strings.HasSuffix(path, ".tar.gz.gpg") ||
		strings.HasSuffix(path, ".tgz.gpg")
}

// ParseTarOptions determines TAR options from file extension
func ParseTarOptions(path string) TarOptions {
	path = strings.ToLower(path)
	
	options := TarOptions{
		Compression: strings.Contains(path, ".gz") || strings.Contains(path, ".tgz"),
		GPGEncrypt:  strings.HasSuffix(path, ".gpg"),
		GPGSign:     false, // This should be set explicitly by user
	}

	return options
}