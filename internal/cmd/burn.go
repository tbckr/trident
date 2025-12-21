package cmd

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

func NewBurnCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "burn",
		Short: "Self-cleanup: Remove configuration and sensitive data",
		Long: `The burn command securely removes Trident's configuration, logs, and cache. 
It also attempts to self-delete the binary (unsupported on Windows).

OpSec Note: Files are overwritten with random data once before deletion.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Print("🔱 This will PERMANENTLY delete your Trident configuration and artifacts. Are you sure? (y/N): ")
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			// Find config dir
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			configDir := filepath.Join(home, ".config", "trident")

			if _, err := os.Stat(configDir); err == nil {
				// Securely wipe config directory
				err = filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() {
						if err := secureWipe(path); err != nil {
							return err
						}
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("failed to wipe config directory: %w", err)
				}

				err = os.RemoveAll(configDir)
				if err != nil {
					return fmt.Errorf("failed to remove config directory: %w", err)
				}
				fmt.Println("✓ Configuration directory removed.")
			} else {
				fmt.Println("- No configuration directory found.")
			}

			// Binary self-deletion
			if runtime.GOOS == "windows" {
				fmt.Println("! Binary self-deletion is not supported on Windows. Please remove it manually.")
			} else {
				exePath, err := os.Executable()
				if err != nil {
					fmt.Printf("! Failed to find executable path: %v\n", err)
				} else {
					if err := secureWipe(exePath); err != nil {
						fmt.Printf("! Failed to wipe binary: %v\n", err)
					}
					if err := os.Remove(exePath); err != nil {
						fmt.Printf("! Failed to delete binary: %v\n", err)
					} else {
						fmt.Println("✓ Trident binary successfully self-deleted.")
					}
				}
			}

			fmt.Println("🔱 Cleanup complete.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal without confirmation")

	return cmd
}

func secureWipe(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Overwrite with random data
	_, err = io.CopyN(file, rand.Reader, info.Size())
	if err != nil && err != io.EOF {
		return err
	}

	return file.Sync()
}
