package cache

import (
	"context"
	"time"
)

type Cache[T any] interface {
	Get(ctx context.Context, key string) (T, bool, error)
	Set(ctx context.Context, key string, value T, expiration time.Duration) error
	Delete(ctx context.Context, key ...string) error
	Name() string
}

type Closer interface {
	Close() error
}
