package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"openapi-mock/pkg/mm"
	"openapi-mock/pkg/recorder"
)

func TestMockRuntimeResetRebuildsHandler(t *testing.T) {
	var buildCount atomic.Int32
	rt := newMockRuntime("127.0.0.1:0", time.Second, func() (http.Handler, error) {
		ver := buildCount.Add(1)
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprintf(w, "v%d", ver)
		}), nil
	})
	if err := rt.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	t.Cleanup(func() { _ = rt.Stop(context.Background()) })

	resp1, err := http.Get("http://" + rt.Addr())
	if err != nil {
		t.Fatalf("request before reset failed: %v", err)
	}
	defer resp1.Body.Close()
	buf1 := make([]byte, 2)
	_, _ = resp1.Body.Read(buf1)

	if err := rt.Reset(context.Background()); err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	resp2, err := http.Get("http://" + rt.Addr())
	if err != nil {
		t.Fatalf("request after reset failed: %v", err)
	}
	defer resp2.Body.Close()
	buf2 := make([]byte, 2)
	_, _ = resp2.Body.Read(buf2)

	if string(buf1) == string(buf2) {
		t.Fatalf("expected different handler versions before/after reset, got %q and %q", string(buf1), string(buf2))
	}
}

func TestResetCallbackClearsStores(t *testing.T) {
	rec := recorder.New()
	rec.Record(recorder.CallRecord{RequestID: "a", Method: "GET /x", Timestamp: time.Now()})
	values := mm.NewStore()
	values.Replace("a", map[string]any{"k": 1})

	called := false
	cb := resetCallback(func(context.Context) error {
		called = true
		return nil
	}, rec, values)

	if err := cb(context.Background()); err != nil {
		t.Fatalf("reset callback failed: %v", err)
	}
	if !called {
		t.Fatalf("expected runtime reset callback to be called")
	}
	if len(rec.GetRecords()) != 0 {
		t.Fatalf("expected recorder to be cleared")
	}
	if len(values.GetAll()) != 0 {
		t.Fatalf("expected context values store to be cleared")
	}
}

func TestMockRuntimeResetIsSerialized(t *testing.T) {
	var buildMu sync.Mutex
	inBuild := 0
	maxInBuild := 0
	rt := newMockRuntime("127.0.0.1:0", time.Second, func() (http.Handler, error) {
		buildMu.Lock()
		inBuild++
		if inBuild > maxInBuild {
			maxInBuild = inBuild
		}
		buildMu.Unlock()

		time.Sleep(10 * time.Millisecond)

		buildMu.Lock()
		inBuild--
		buildMu.Unlock()
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), nil
	})
	if err := rt.Start(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	t.Cleanup(func() { _ = rt.Stop(context.Background()) })

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := rt.Reset(context.Background()); err != nil {
				t.Errorf("reset failed: %v", err)
			}
		}()
	}
	wg.Wait()

	if maxInBuild > 1 {
		t.Fatalf("expected serialized reset builds, max concurrent builds=%d", maxInBuild)
	}
}
