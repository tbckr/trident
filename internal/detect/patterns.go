package detect

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/tbckr/trident/internal/appdir"
)

//go:embed patterns.yaml
var embeddedPatterns []byte

// CDNPattern maps a CNAME suffix to a CDN provider name.
type CDNPattern struct {
	Suffix   string `yaml:"suffix"`
	Provider string `yaml:"provider"`
}

// EmailPattern maps an MX exchange suffix to an email provider name.
type EmailPattern struct {
	Suffix   string `yaml:"suffix"`
	Provider string `yaml:"provider"`
}

// DNSPattern maps an NS server suffix or substring to a DNS hosting provider name.
// If Contains is non-empty, substring matching is used; otherwise suffix matching applies.
type DNSPattern struct {
	Suffix   string `yaml:"suffix"`
	Contains string `yaml:"contains"`
	Provider string `yaml:"provider"`
}

// TXTPattern maps a TXT record substring to a provider name and service type.
type TXTPattern struct {
	Substring string      `yaml:"substring"`
	Provider  string      `yaml:"provider"`
	Type      ServiceType `yaml:"type"`
}

// Patterns holds all detection patterns for CDN, email, DNS, and TXT records.
type Patterns struct {
	CDN   []CDNPattern   `yaml:"cdn"`
	Email []EmailPattern `yaml:"email"`
	DNS   []DNSPattern   `yaml:"dns"`
	TXT   []TXTPattern   `yaml:"txt"`
}

// LoadPatterns tries each path in order; the first file that exists is used.
// Falls back to the embedded patterns.yaml when no override file is found.
func LoadPatterns(paths ...string) (Patterns, error) {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Patterns{}, fmt.Errorf("reading patterns file %q: %w", path, err)
		}
		var p Patterns
		if err := yaml.Unmarshal(data, &p); err != nil {
			return Patterns{}, fmt.Errorf("parsing patterns file %q: %w", path, err)
		}
		return p, nil
	}
	// No override file found — fall back to the embedded defaults.
	var p Patterns
	if err := yaml.Unmarshal(embeddedPatterns, &p); err != nil {
		return Patterns{}, fmt.Errorf("parsing embedded patterns: %w", err)
	}
	return p, nil
}

// DefaultPatternPaths returns the two override paths in priority order:
// user-edited file first, then the reserved download path.
// Derives from appdir.ConfigDir().
func DefaultPatternPaths() ([]string, error) {
	dir, err := appdir.ConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolving config dir: %w", err)
	}
	return []string{
		filepath.Join(dir, "detect.yaml"),
		filepath.Join(dir, "detect-downloaded.yaml"),
	}, nil
}
