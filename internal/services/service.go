package services

import "context"

// Service is the contract every Trident service must implement.
type Service interface {
	Name() string
	Run(ctx context.Context, input string) (any, error)
}
