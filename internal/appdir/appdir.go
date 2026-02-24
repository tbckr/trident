package appdir

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigDir returns the OS-specific config directory for trident.
// Linux: $XDG_CONFIG_HOME/trident  macOS: ~/Library/Application Support/trident
// Windows: %AppData%/trident
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config dir: %w", err)
	}
	return filepath.Join(base, "trident"), nil
}

// EnsureFile creates path and its parent directories if they do not exist.
// The file is created with 0600 permissions (owner read/write only).
// A no-op if the file already exists.
func EnsureFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("creating config file: %w", err)
	}
	return f.Close()
}
