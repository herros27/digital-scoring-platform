package cache

import (
	"bytes"
	"net/http"
	"strings"
	"time"

	"github.com/maypok86/otter"
)

// Item represents a cached HTTP response.
type Item struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

// Cache wraps an Otter in-memory cache configured with variable TTL and S3-FIFO eviction.
// ponytail: in-memory per-instance cache. Ceiling: single process memory, not shared across replicas.
// Upgrade path: Redis cluster when moving to multi-server distributed deployment.
type Cache struct {
	inner otter.CacheWithVariableTTL[string, *Item]
}

// New creates a new Otter-backed in-memory cache with specified capacity.
func New(capacity int) (*Cache, error) {
	if capacity <= 0 {
		capacity = 10_000
	}
	c, err := otter.MustBuilder[string, *Item](capacity).
		WithVariableTTL().
		Build()
	if err != nil {
		return nil, err
	}
	return &Cache{inner: c}, nil
}

// MustNew creates a new Cache or panics on failure.
func MustNew(capacity int) *Cache {
	c, err := New(capacity)
	if err != nil {
		panic(err)
	}
	return c
}

// Get retrieves an item by key.
func (c *Cache) Get(key string) (*Item, bool) {
	return c.inner.Get(key)
}

// Set stores an item with a given TTL.
func (c *Cache) Set(key string, item *Item, ttl time.Duration) {
	c.inner.Set(key, item, ttl)
}

// Delete removes a key from cache.
func (c *Cache) Delete(key string) {
	c.inner.Delete(key)
}

// InvalidateCompetition purges all cached entries belonging to a competition ID.
// ponytail: scans keys containing compID using DeleteByFunc. Bounded by capacity (10k items), completes in microseconds.
func (c *Cache) InvalidateCompetition(compID string) {
	if compID == "" {
		return
	}
	c.inner.DeleteByFunc(func(key string, _ *Item) bool {
		return strings.Contains(key, compID)
	})
}

// Clear flushes all entries in cache.
func (c *Cache) Clear() {
	c.inner.Clear()
}

// Close closes the underlying Otter cache.
func (c *Cache) Close() {
	c.inner.Close()
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
	body          bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	if !r.headerWritten {
		r.statusCode = code
		r.headerWritten = true
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

// Handler wraps an http.HandlerFunc with Otter in-memory caching.
// Only successful GET requests (HTTP 200) are cached for the duration of ttl.
func (c *Cache) Handler(ttl time.Duration, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET or HEAD requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next(w, r)
			return
		}

		key := r.Method + ":" + r.URL.RequestURI()

		if item, ok := c.Get(key); ok {
			if item.ContentType != "" {
				w.Header().Set("Content-Type", item.ContentType)
			}
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(item.StatusCode)
			w.Write(item.Body)
			return
		}

		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next(rec, r)

		if rec.statusCode == http.StatusOK {
			contentType := rec.Header().Get("Content-Type")
			c.Set(key, &Item{
				StatusCode:  rec.statusCode,
				ContentType: contentType,
				Body:        rec.body.Bytes(),
			}, ttl)
		}

		w.Header().Set("X-Cache", "MISS")
		w.WriteHeader(rec.statusCode)
		w.Write(rec.body.Bytes())
	}
}
