package config

import "fmt"

type ApiKeyNotSetError struct {
	Plugin string
}

func (e *ApiKeyNotSetError) Error() string {
	return fmt.Sprintf("API key not set for plugin %s", e.Plugin)
}
