package cache

import (
	"context"
	"github.com/pkg/errors"
	"time"
)

type MultilayerCache[T any] struct {
	caches     []Cache[T]
	expiration time.Duration
	zero       T
}

// NewMultilayerCache
// @expiration expiration date (of document)
// @caches cache implementation
func NewMultilayerCache[T any](expiration time.Duration, caches ...Cache[T]) *MultilayerCache[T] {
	return &MultilayerCache[T]{
		expiration: expiration,
		caches:     caches,
	}
}

func (mc *MultilayerCache[T]) Set(ctx context.Context, key string, value T) error {
	for _, cache := range mc.caches {
		err := cache.Set(ctx, key, value, mc.expiration)
		if err != nil {
			return errors.Errorf("cache:%v  error: %v", cache.Name(), err)
		}
	}
	return nil
}

func (mc *MultilayerCache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	var result T
	var index int
	var exist bool
	for i, cache := range mc.caches {
		v, ok, err := cache.Get(ctx, key)
		if err != nil {
			return v, ok, errors.Errorf("cache:%v  error: %v", cache.Name(), err)
		}
		if ok {
			index = i
			result = v
			exist = true
			break
		}
	}
	if index != 0 && exist {
		// If the previous layer does not have a value for the key then set the
		for _, c := range mc.caches[:index] {
			_ = c.Set(ctx, key, result, mc.expiration)
		}
		return result, true, nil
	}
	return mc.zero, false, nil
}

func (mc *MultilayerCache[T]) Close() {
	for _, cache := range mc.caches {
		if c, ok := cache.(Closer); ok {
			_ = c.Close()
		}
	}
}
