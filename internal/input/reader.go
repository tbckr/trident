package input

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func GetInputs(args []string, stdin io.Reader) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}

	var inputs []string
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	// Check if data is being piped in
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				inputs = append(inputs, line)
			}
		}
		if err = scanner.Err(); err != nil {
			return nil, err
		}
	}

	return inputs, nil
}
