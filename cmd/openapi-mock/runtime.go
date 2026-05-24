package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type runtimeBuilder func() (http.Handler, error)

type mockRuntime struct {
	mu              sync.Mutex
	addr            string
	shutdownTimeout time.Duration
	build           runtimeBuilder

	server   *http.Server
	listener net.Listener
}

func newMockRuntime(addr string, shutdownTimeout time.Duration, build runtimeBuilder) *mockRuntime {
	return &mockRuntime{addr: addr, shutdownTimeout: shutdownTimeout, build: build}
}

func (r *mockRuntime) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.startLocked()
}

func (r *mockRuntime) Reset(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.stopLocked(ctx); err != nil {
		return err
	}
	return r.startLocked()
}

func (r *mockRuntime) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopLocked(ctx)
}

func (r *mockRuntime) Addr() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listener == nil {
		return ""
	}
	return r.listener.Addr().String()
}

func (r *mockRuntime) startLocked() error {
	if r.server != nil {
		return nil
	}
	h, err := r.build()
	if err != nil {
		return fmt.Errorf("build handler: %w", err)
	}
	ln, err := net.Listen("tcp", r.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", r.addr, err)
	}
	srv := &http.Server{Handler: h}
	r.server = srv
	r.listener = ln
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("mock runtime server error: %v", err)
		}
	}()
	return nil
}

func (r *mockRuntime) stopLocked(ctx context.Context) error {
	if r.server == nil {
		return nil
	}
	shutdownCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(ctx, r.shutdownTimeout)
		defer cancel()
	}
	err := r.server.Shutdown(shutdownCtx)
	r.server = nil
	r.listener = nil
	if err != nil {
		return fmt.Errorf("shutdown mock runtime: %w", err)
	}
	return nil
}
