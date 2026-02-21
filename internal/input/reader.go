package input

import (
	"bufio"
	"io"
	"strings"
)

// Read reads lines from r, trims whitespace, and returns non-empty lines.
// Blank lines and lines that are only whitespace are dropped.
func Read(r io.Reader) ([]string, error) {
	var inputs []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			inputs = append(inputs, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return inputs, nil
}
