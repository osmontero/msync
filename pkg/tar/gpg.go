package tar

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// GPGHandler handles GPG encryption, decryption, signing, and verification
type GPGHandler struct {
	KeyID     string
	KeyringPath string
}

// NewGPGHandler creates a new GPG handler
func NewGPGHandler(keyringPath, keyID string) (*GPGHandler, error) {
	handler := &GPGHandler{
		KeyID:       keyID,
		KeyringPath: keyringPath,
	}

	// Verify GPG is available
	if err := handler.checkGPGAvailable(); err != nil {
		return nil, err
	}

	// Verify key exists if specified
	if keyID != "" {
		if err := handler.verifyKey(keyID); err != nil {
			return nil, fmt.Errorf("GPG key verification failed: %w", err)
		}
	}

	return handler, nil
}

// checkGPGAvailable verifies that GPG is installed and available
func (g *GPGHandler) checkGPGAvailable() error {
	cmd := exec.Command("gpg", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("GPG is not available: %w", err)
	}
	return nil
}

// verifyKey checks if the specified GPG key exists
func (g *GPGHandler) verifyKey(keyID string) error {
	args := []string{"--list-keys", keyID}
	if g.KeyringPath != "" {
		args = append([]string{"--keyring", g.KeyringPath}, args...)
	}

	cmd := exec.Command("gpg", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("key %s not found", keyID)
	}
	return nil
}

// Encrypt encrypts data using GPG and returns a WriteCloser
func (g *GPGHandler) Encrypt(writer io.Writer) (io.WriteCloser, error) {
	if g.KeyID == "" {
		return nil, fmt.Errorf("no GPG key ID specified for encryption")
	}

	args := []string{
		"--cipher-algo", "AES256",
		"--compress-algo", "2",
		"--digest-algo", "SHA256",
		"--encrypt",
		"--armor",
		"--recipient", g.KeyID,
	}

	if g.KeyringPath != "" {
		args = append([]string{"--keyring", g.KeyringPath}, args...)
	}

	cmd := exec.Command("gpg", args...)
	cmd.Stdout = writer

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to start GPG encryption: %w", err)
	}

	return &gpgWriter{
		cmd:   cmd,
		stdin: stdin,
	}, nil
}

// Decrypt decrypts GPG-encrypted data
func (g *GPGHandler) Decrypt(reader io.Reader) (io.Reader, error) {
	args := []string{
		"--decrypt",
		"--quiet",
	}

	if g.KeyringPath != "" {
		args = append([]string{"--keyring", g.KeyringPath}, args...)
	}

	cmd := exec.Command("gpg", args...)
	cmd.Stdin = reader

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start GPG decryption: %w", err)
	}

	return &gpgReader{
		cmd:    cmd,
		stdout: stdout,
	}, nil
}

// Sign creates a detached GPG signature for a file
func (g *GPGHandler) Sign(filePath, signaturePath string) error {
	if g.KeyID == "" {
		return fmt.Errorf("no GPG key ID specified for signing")
	}

	args := []string{
		"--detach-sign",
		"--armor",
		"--output", signaturePath,
		"--local-user", g.KeyID,
		filePath,
	}

	if g.KeyringPath != "" {
		args = append([]string{"--keyring", g.KeyringPath}, args...)
	}

	cmd := exec.Command("gpg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("GPG signing failed: %w, output: %s", err, string(output))
	}

	return nil
}

// Verify verifies a detached GPG signature
func (g *GPGHandler) Verify(filePath, signaturePath string) error {
	args := []string{
		"--verify",
		signaturePath,
		filePath,
	}

	if g.KeyringPath != "" {
		args = append([]string{"--keyring", g.KeyringPath}, args...)
	}

	cmd := exec.Command("gpg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("GPG signature verification failed: %w, output: %s", err, string(output))
	}

	return nil
}

// IsEncrypted checks if data appears to be GPG encrypted
func (g *GPGHandler) IsEncrypted(data []byte) bool {
	// Check for ASCII armored GPG data
	if bytes.Contains(data, []byte("-----BEGIN PGP MESSAGE-----")) {
		return true
	}

	// Check for binary GPG data (starts with specific bytes)
	if len(data) >= 2 && data[0] == 0x85 {
		return true // GPG binary format
	}

	return false
}

// ListKeys lists available GPG keys
func (g *GPGHandler) ListKeys() ([]string, error) {
	args := []string{"--list-keys", "--with-colons"}
	if g.KeyringPath != "" {
		args = append([]string{"--keyring", g.KeyringPath}, args...)
	}

	cmd := exec.Command("gpg", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list GPG keys: %w", err)
	}

	var keys []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "pub:") {
			parts := strings.Split(line, ":")
			if len(parts) > 4 && parts[4] != "" {
				keys = append(keys, parts[4])
			}
		}
	}

	return keys, nil
}

// gpgWriter wraps a GPG encryption process
type gpgWriter struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func (w *gpgWriter) Write(p []byte) (n int, err error) {
	return w.stdin.Write(p)
}

func (w *gpgWriter) Close() error {
	if err := w.stdin.Close(); err != nil {
		return err
	}
	return w.cmd.Wait()
}

// gpgReader wraps a GPG decryption process
type gpgReader struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

func (r *gpgReader) Read(p []byte) (n int, err error) {
	n, err = r.stdout.Read(p)
	if err == io.EOF {
		// Wait for process to complete
		if waitErr := r.cmd.Wait(); waitErr != nil {
			return n, fmt.Errorf("GPG process failed: %w", waitErr)
		}
	}
	return n, err
}