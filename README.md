# Multi-layer caching implemented by golang

### example
#### local caeche
```go
package cache

import (
	"context"
	"time"
)

type LocalCache[T any] struct {
	name  string
	cache map[string]T
	zero  T
}

var _ Cache[any] = (*LocalCache[any])(nil)

// NewLocalCache
// @name cache name 
// @defaultExpiration default cache key expiration time
// @cleanupInterval default time between cache cleanups
func NewLocalCache[T any](name string, defaultExpiration, cleanupInterval time.Duration) Cache[T] {
	return &LocalCache[T]{
		name:  name,
		cache: make(map[string]T),
	}
}

func (lc *LocalCache[T]) Get(_ context.Context, key string) (T, bool, error) {
	v, exist := lc.cache[key]
	if !exist {
		return lc.zero, false, nil
	}
	vv, _ := v.(T)
	return vv, exist, nil
}

func (lc *LocalCache[T]) Set(_ context.Context, key string, t T, _ time.Duration) error {
	lc.cache[key] = t
	return nil
}

func (lc *LocalCache[T]) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(lc.cache, key)
	}
	return nil
}

func (lc *LocalCache[T]) Name() string {
	return lc.name
}
```
#### remote cache
```go
package cache

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"reflect"
	"time"
)
type RedisClient struct {
	*redis.Client
}

func NewRedisClient(ctx context.Context, options *redis.Options) (*RedisClient, error) {
	rc := &RedisClient{redis.NewClient(options)}
	return rc, rc.Ping(ctx).Err()
}

func (rc *RedisClient) Close() error {
	return rc.Client.Close()
}

// RedisCache
// Redis Cache the implementation only supports the normal key/value format
type RedisCache[T any] struct {
	name  string
	cache *RedisClient
	zero  T
}

func NewRedisCache[T any](name string, rc *RedisClient) Cache[T] {
	return &RedisCache[T]{
		name:  name,
		cache: rc,
	}
}

func (rc *RedisCache[T]) Name() string {
	return rc.name
}

func (rc *RedisCache[T]) Get(ctx context.Context, key string) (T, bool, error) {
	strCmd := rc.cache.Get(ctx, key)
	value, err := strCmd.Result()
	if err != nil {
		if err == redis.Nil {
			return rc.zero, false, nil
		}
		return rc.zero, false, err
	}
	var v T
	switch reflect.TypeOf(rc.zero).Kind() {
	case reflect.String:
		var ptr any
		ptr = value
		v = ptr.(T)
	default:
		err = json.Unmarshal([]byte(value), &v)
		if err != nil {
			return rc.zero, false, err
		}
	}
	return v, true, err
}

func (rc *RedisCache[T]) Set(ctx context.Context, key string, t T, expiration time.Duration) error {
	var v string
	switch reflect.TypeOf(t).Kind() {
	case reflect.String:
		v = fmt.Sprintf("%v", t)
	default:
		body, err := json.Marshal(t)
		if err != nil {
			return err
		}
		v = string(body)
	}
	statusCmd := rc.cache.Set(ctx, key, v, expiration)
	return statusCmd.Err()
}

func (rc *RedisCache[T]) Delete(ctx context.Context, keys ...string) error {
	intCmd := rc.cache.Del(ctx, keys...)
	return intCmd.Err()
}

func (rc *RedisCache[T]) Close() error {
	return rc.cache.Close()
}
```
#### use
```go
ctx:=context.Background()
local := NewLocalCache[string]("local", 1*time.Minute, 1*time.Minute)
rc, err := NewRedisClient(ctx, &redis.Options{
	Addr:     "",
	Password: "",
})
if err!=nil{
	panic(err)
}
remote := NewRedisCache[string]("remote", rc)
mCache := NewMultilayerCache[string](1*time.Minute, local, remote)
defer mCache.Close()

err=mCache.Set(ctx,"key","value")
if err!=nil{
	panic(err)
}
fmt.Println(mCache.Get(ctx,"key"))
```