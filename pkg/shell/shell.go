package shell

import "os"

// PipedShell checks, if the shell is piped
func PipedShell() (bool, error) {
	return PipedShellWithFile(os.Stdin)
}

func PipedShellWithFile(stdin *os.File) (bool, error) {
	fi, err := stdin.Stat()
	if err != nil {
		return false, err
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return false, nil
	}
	return true, nil
}
