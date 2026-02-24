package delivery

import "context"

// Delivery defines the interface for sending a compiled digest.
type Delivery interface {
	Name() string
	Send(ctx context.Context, digest string) error
}
