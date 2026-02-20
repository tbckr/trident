// Package pgp provides a service for searching PGP keys via the HKP protocol.
package pgp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/imroc/req/v3"

	"github.com/tbckr/trident/internal/output"
	"github.com/tbckr/trident/internal/pap"
	"github.com/tbckr/trident/internal/services"
)

const hkpURL = "https://keys.openpgp.org/pks/lookup?op=index&search=%s&options=mr"

// Key represents a single PGP key from a keyserver query.
type Key struct {
	KeyID     string   `json:"key_id"`
	Algorithm string   `json:"algorithm"`
	Bits      int      `json:"bits"`
	CreatedAt string   `json:"created_at"`
	ExpiresAt string   `json:"expires_at,omitempty"`
	UIDs      []string `json:"uids"`
}

// Result holds the output of a PGP key search.
type Result struct {
	Input string `json:"input"`
	Keys  []Key  `json:"keys"`
}

// IsEmpty returns true when no keys were found.
func (r *Result) IsEmpty() bool {
	return len(r.Keys) == 0
}

// WriteText writes a human-readable table to w.
// Each key is rendered with its UIDs on separate rows.
func (r *Result) WriteText(w io.Writer) error {
	tbl := output.NewWrappingTable(w, 20, 30)
	tbl.Header([]string{"Key ID", "UID", "Algorithm", "Bits", "Created", "Expires"})
	rows := make([][]string, 0, len(r.Keys))
	for _, k := range r.Keys {
		uid := strings.Join(k.UIDs, ", ")
		rows = append(rows, []string{k.KeyID, uid, k.Algorithm, strconv.Itoa(k.Bits), k.CreatedAt, k.ExpiresAt})
	}
	if err := tbl.Bulk(rows); err != nil {
		return err
	}
	return tbl.Render()
}

// WritePlain writes one line per key: "<keyid> <first_uid>" to w.
func (r *Result) WritePlain(w io.Writer) error {
	for _, k := range r.Keys {
		uid := ""
		if len(k.UIDs) > 0 {
			uid = k.UIDs[0]
		}
		if _, err := fmt.Fprintf(w, "%s %s\n", k.KeyID, uid); err != nil {
			return err
		}
	}
	return nil
}

// Service queries a HKP keyserver for PGP keys.
type Service struct {
	client *req.Client
	logger *slog.Logger
}

// NewService creates a new PGP Service.
func NewService(client *req.Client, logger *slog.Logger) *Service {
	return &Service{client: client, logger: logger}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "pgp" }

// PAP returns the PAP classification for this service.
func (s *Service) PAP() pap.Level { return pap.AMBER }

// AggregateResults combines multiple PGP results into a MultiResult.
func (s *Service) AggregateResults(results []services.Result) services.Result {
	mr := &MultiResult{}
	for _, r := range results {
		mr.Results = append(mr.Results, r.(*Result))
	}
	return mr
}

// Run searches for PGP keys matching the given query (email or name).
func (s *Service) Run(ctx context.Context, input string) (services.Result, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("%w: query must not be empty", services.ErrInvalidInput)
	}

	query := url.QueryEscape(input)
	reqURL := fmt.Sprintf(hkpURL, query)

	resp, err := s.client.R().SetContext(ctx).Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", services.ErrRequestFailed, err)
	}
	if resp.StatusCode == 404 {
		// Key not found — return empty result, not an error.
		return &Result{Input: input}, nil
	}
	if !resp.IsSuccessState() {
		body := resp.String()
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		return nil, fmt.Errorf("%w: HTTP %d: %q", services.ErrRequestFailed, resp.StatusCode, body)
	}

	keys, err := parseMRINDEX(resp.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse HKP response: %w", err)
	}

	return &Result{Input: input, Keys: keys}, nil
}

// parseMRINDEX parses HKP machine-readable index format into a slice of Keys.
//
// Format:
//
//	info:1:N
//	pub:<keyid>:<algo>:<bits>:<created>:<expires>:<flags>
//	uid:<uid>:<created>:<expires>:<flags>
//	...
func parseMRINDEX(body string) ([]Key, error) {
	var keys []Key
	var current *Key

	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "info":
			// info:version:count — nothing to do.
		case "pub":
			// pub:<keyid>:<algo>:<bits>:<created>:<expires>:<flags>
			if current != nil {
				keys = append(keys, *current)
			}
			current = &Key{}
			if len(parts) > 1 {
				current.KeyID = output.StripANSI(parts[1])
			}
			if len(parts) > 2 {
				current.Algorithm = algoName(parts[2])
			}
			if len(parts) > 3 {
				if bits, err := strconv.Atoi(parts[3]); err == nil {
					current.Bits = bits
				}
			}
			if len(parts) > 4 {
				current.CreatedAt = formatUnix(parts[4])
			}
			if len(parts) > 5 {
				current.ExpiresAt = formatUnix(parts[5])
			}
		case "uid":
			// uid:<uid>:<created>:<expires>:<flags>
			if current != nil && len(parts) > 1 {
				uid := output.StripANSI(parts[1])
				if uid != "" {
					current.UIDs = append(current.UIDs, uid)
				}
			}
		}
	}
	if current != nil {
		keys = append(keys, *current)
	}
	return keys, scanner.Err()
}

// algoName translates an HKP algorithm number to a human-readable name.
func algoName(code string) string {
	switch code {
	case "1", "2", "3":
		return "RSA"
	case "17":
		return "DSA"
	case "18":
		return "ECDH"
	case "19":
		return "ECDSA"
	case "22":
		return "EdDSA"
	default:
		return code
	}
}

// formatUnix converts a Unix timestamp string to YYYY-MM-DD, or returns "" if empty/invalid.
func formatUnix(ts string) string {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return ""
	}
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return ts
	}
	return time.Unix(n, 0).UTC().Format("2006-01-02")
}
