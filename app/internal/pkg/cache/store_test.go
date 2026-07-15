package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T, cache *MemoryCache)
	}{
		{
			name: "returns cache miss for unknown key",
			run: func(t *testing.T, cache *MemoryCache) {
				_, err := cache.Get(context.Background(), "missing")
				if !errors.Is(err, ErrCacheMiss) {
					t.Fatalf("Get() error = %v, want ErrCacheMiss", err)
				}
			},
		},
		{
			name: "stores defensive value copies",
			run: func(t *testing.T, cache *MemoryCache) {
				ctx := context.Background()
				input := []byte("value")
				if err := cache.Set(ctx, "key", input, 0); err != nil {
					t.Fatalf("Set() error = %v", err)
				}
				input[0] = 'X'

				first, err := cache.Get(ctx, "key")
				if err != nil {
					t.Fatalf("Get() error = %v", err)
				}
				first[0] = 'Y'

				second, err := cache.Get(ctx, "key")
				if err != nil {
					t.Fatalf("Get() second error = %v", err)
				}
				if got := string(second); got != "value" {
					t.Fatalf("Get() value = %q, want %q", got, "value")
				}
			},
		},
		{
			name: "deletes existing key",
			run: func(t *testing.T, cache *MemoryCache) {
				ctx := context.Background()
				if err := cache.Set(ctx, "key", []byte("value"), 0); err != nil {
					t.Fatalf("Set() error = %v", err)
				}
				if err := cache.Del(ctx, "key"); err != nil {
					t.Fatalf("Del() error = %v", err)
				}
				if _, err := cache.Get(ctx, "key"); !errors.Is(err, ErrCacheMiss) {
					t.Fatalf("Get() after Del() error = %v, want ErrCacheMiss", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cache := NewMemoryCache(0)
			t.Cleanup(cache.Stop)
			tt.run(t, cache)
		})
	}
}

func TestMemoryCacheTTLDoesNotExtendOnGet(t *testing.T) {
	t.Parallel()

	cache := NewMemoryCache(5 * time.Millisecond)
	t.Cleanup(cache.Stop)
	ctx := context.Background()
	if err := cache.Set(ctx, "key", []byte("value"), 80*time.Millisecond); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if _, err := cache.Get(ctx, "key"); err != nil {
		t.Fatalf("Get() before expiry error = %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if _, err := cache.Get(ctx, "key"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("Get() after expiry error = %v, want ErrCacheMiss", err)
	}
}

func TestMemoryCacheCapacityEvictsLeastRecentlyUsed(t *testing.T) {
	t.Parallel()

	cache := NewMemoryCache(0, WithMemoryCacheCapacity(2))
	t.Cleanup(cache.Stop)
	ctx := context.Background()
	for _, key := range []string{"first", "second"} {
		if err := cache.Set(ctx, key, []byte(key), 0); err != nil {
			t.Fatalf("Set(%q) error = %v", key, err)
		}
	}
	if _, err := cache.Get(ctx, "first"); err != nil {
		t.Fatalf("Get(first) error = %v", err)
	}
	if err := cache.Set(ctx, "third", []byte("third"), 0); err != nil {
		t.Fatalf("Set(third) error = %v", err)
	}

	if _, err := cache.Get(ctx, "second"); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("Get(second) error = %v, want ErrCacheMiss", err)
	}
	for _, key := range []string{"first", "third"} {
		if _, err := cache.Get(ctx, key); err != nil {
			t.Fatalf("Get(%q) error = %v", key, err)
		}
	}
}

func TestMemoryCacheStopIsConcurrentSafe(t *testing.T) {
	t.Parallel()

	cache := NewMemoryCache(time.Hour)
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Stop()
		}()
	}
	wg.Wait()
}

func TestMemoryCacheWithContextStopsJanitor(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cache := NewMemoryCacheWithContext(ctx, time.Hour)
	cancel()

	select {
	case <-cache.done:
	case <-time.After(time.Second):
		t.Fatal("janitor did not stop after context cancellation")
	}
	cache.Stop()
}
