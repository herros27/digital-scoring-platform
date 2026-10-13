package cache

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestCache_BasicOperations(t *testing.T) {
	c, err := New(100)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer c.Close()

	c.Set("test-key", &Item{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"status":"ok"}`),
	}, 1*time.Minute)

	item, ok := c.Get("test-key")
	if !ok {
		t.Fatal("expected item to be found in cache")
	}
	if item.StatusCode != 200 || string(item.Body) != `{"status":"ok"}` {
		t.Fatalf("unexpected cached item: %+v", item)
	}

	c.Delete("test-key")
	_, ok = c.Get("test-key")
	if ok {
		t.Fatal("expected item to be deleted from cache")
	}
}

func TestCache_InvalidateCompetition(t *testing.T) {
	c := MustNew(100)
	defer c.Close()

	compID := "comp-uuid-123"
	key1 := "GET:/api/v1/competitions/" + compID + "/report/summary"
	key2 := "GET:/api/v1/competitions/" + compID + "/report/medals"
	keyOther := "GET:/api/v1/competitions/other-uuid-999/report/medals"

	c.Set(key1, &Item{StatusCode: 200, Body: []byte("1")}, time.Minute)
	c.Set(key2, &Item{StatusCode: 200, Body: []byte("2")}, time.Minute)
	c.Set(keyOther, &Item{StatusCode: 200, Body: []byte("3")}, time.Minute)

	c.InvalidateCompetition(compID)

	if _, ok := c.Get(key1); ok {
		t.Errorf("expected key1 to be invalidated")
	}
	if _, ok := c.Get(key2); ok {
		t.Errorf("expected key2 to be invalidated")
	}
	if _, ok := c.Get(keyOther); !ok {
		t.Errorf("expected keyOther to still exist")
	}
}

func TestCache_HandlerWrapper(t *testing.T) {
	c := MustNew(100)
	defer c.Close()

	var counter int64
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		val := atomic.AddInt64(&counter, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"count":%d}`, val)
	}

	cachedHandler := c.Handler(500*time.Millisecond, handlerFunc)

	// First request: MISS (counter = 1)
	req1 := httptest.NewRequest("GET", "/api/v1/report/medals", nil)
	rec1 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("req1: expected 200, got %d", rec1.Code)
	}
	if rec1.Header().Get("X-Cache") != "MISS" {
		t.Errorf("req1: expected X-Cache: MISS, got %s", rec1.Header().Get("X-Cache"))
	}
	if rec1.Body.String() != `{"count":1}` {
		t.Errorf("req1: expected body count:1, got %s", rec1.Body.String())
	}

	// Second request: HIT (counter should STILL be 1)
	req2 := httptest.NewRequest("GET", "/api/v1/report/medals", nil)
	rec2 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("req2: expected 200, got %d", rec2.Code)
	}
	if rec2.Header().Get("X-Cache") != "HIT" {
		t.Errorf("req2: expected X-Cache: HIT, got %s", rec2.Header().Get("X-Cache"))
	}
	if rec2.Body.String() != `{"count":1}` {
		t.Errorf("req2: expected body count:1, got %s", rec2.Body.String())
	}

	// Third request with different query param: MISS
	req3 := httptest.NewRequest("GET", "/api/v1/report/medals?team=garuda", nil)
	rec3 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(rec3, req3)

	if rec3.Header().Get("X-Cache") != "MISS" {
		t.Errorf("req3: expected X-Cache: MISS for new query param, got %s", rec3.Header().Get("X-Cache"))
	}
	if rec3.Body.String() != `{"count":2}` {
		t.Errorf("req3: expected body count:2, got %s", rec3.Body.String())
	}
}

func TestCache_HandlerNon200NotCached(t *testing.T) {
	c := MustNew(100)
	defer c.Close()

	var counter int64
	errorHandler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&counter, 1)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}

	cachedHandler := c.Handler(time.Minute, errorHandler)

	req1 := httptest.NewRequest("GET", "/api/v1/report/unknown", nil)
	rec1 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/report/unknown", nil)
	rec2 := httptest.NewRecorder()
	cachedHandler.ServeHTTP(rec2, req2)

	// Since 404 is not cached, counter must be 2
	if atomic.LoadInt64(&counter) != 2 {
		t.Errorf("expected error not to be cached, counter=%d", counter)
	}
}
func TestInvalidateKeysContaining(t *testing.T) {
	c := MustNew(100)
	item := &Item{StatusCode: 200, Body: []byte("x")}
	for _, k := range []string{
		"GET:/api/v1/public/competitions",
		"GET:/api/v1/public/competitions/abc/brackets",
		"GET:/api/v1/competitions/abc/report/summary",
	} {
		c.Set(k, item, time.Minute)
	}

	c.InvalidateKeysContaining("") // no-op
	if _, ok := c.Get("GET:/api/v1/public/competitions"); !ok {
		t.Fatal("substring kosong tidak boleh menghapus apa pun")
	}

	c.InvalidateKeysContaining("/api/v1/public/competitions")
	for _, k := range []string{"GET:/api/v1/public/competitions", "GET:/api/v1/public/competitions/abc/brackets"} {
		if _, ok := c.Get(k); ok {
			t.Fatalf("%s seharusnya sudah dibuang", k)
		}
	}
	if _, ok := c.Get("GET:/api/v1/competitions/abc/report/summary"); !ok {
		t.Fatal("entri lain tidak boleh ikut terhapus")
	}
}
