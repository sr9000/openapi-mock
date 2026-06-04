package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	baseURL = flag.String("base-url", "http://localhost:8080", "base URL of openapi-mock server")
	// Keep it modest by default. This load generator spawns request goroutines continuously.
	tick = flag.Duration("tick", 100*time.Millisecond, "rate update tick")
)

type scenario struct {
	rpsFunc func(float64) float64
	task    func(context.Context, *http.Client)
}

func main() {
	flag.Parse()

	client := &http.Client{Timeout: 10 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	scenarios := []scenario{
		// Happy path traffic
		{rpsFunc: sineWave(50, 50, 120*time.Second, 0), task: listPetsTask("happy")},
		{rpsFunc: sineWave(45, 35, 70*time.Second, 0), task: createPetTask("Boots-Cats")},
		{rpsFunc: sineWave(30, 30, 126*time.Second, 0), task: getPetTask(1)},

		// Periodic unhandled errors from strict handler error path
		{rpsFunc: sineWave(5, 15, 140*time.Second, 0), task: createPetTask("error: Alice")},
		{rpsFunc: sineWave(3, 13, 134*time.Second, 5*time.Second), task: createPetTask("error: Bob")},
		{rpsFunc: sineWave(7, 10, 88*time.Second, 0), task: getPetTask(500)},
		{rpsFunc: sineWave(2, 5, 133*time.Second, 0), task: getPetTask(404)},

		// Periodic panics caught by middleware
		{rpsFunc: sineWave(2, 10, 120*time.Second, 0), task: createPetTask("panic: 42")},
		{rpsFunc: sineWave(1, 4, 90*time.Second, 30*time.Second), task: deletePetTask(999)},
	}

	for _, s := range scenarios {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			runLoad(ctx, client, *tick, s.rpsFunc, s.task)
		}()
	}

	log.Printf("OpenAPI petstore load generators started (base-url=%s). Press Ctrl+C to stop.", *baseURL)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Printf("Shutting down...")
	cancel()
	wg.Wait()
}

func runLoad(ctx context.Context, c *http.Client, tick time.Duration, rpsFunc func(float64) float64, task func(context.Context, *http.Client)) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	startTime := time.Now()
	var accumulator float64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			elapsed := now.Sub(startTime).Seconds()
			rate := rpsFunc(elapsed)

			// rate is req/sec. interval is tick.Seconds().
			accumulator += rate * tick.Seconds()

			count := int(accumulator)
			accumulator -= float64(count)

			for i := 0; i < count; i++ {
				go task(ctx, c)
			}
		}
	}
}

func sineWave(base, amp float64, period time.Duration, offset time.Duration) func(float64) float64 {
	freq := 1.0 / period.Seconds()
	return func(t float64) float64 {
		val := base + amp*math.Sin(2*math.Pi*freq*(t+offset.Seconds()))
		if val < 0 {
			return 0
		}
		return val
	}
}

func listPetsTask(label string) func(context.Context, *http.Client) {
	return func(ctx context.Context, c *http.Client) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, *baseURL+"/pets", nil)
		resp, err := c.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()

		_ = label
	}
}

func getPetTask(id int64) func(context.Context, *http.Client) {
	return func(ctx context.Context, c *http.Client) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, *baseURL+"/pets/"+itoa(id), nil)
		resp, err := c.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}
}

func deletePetTask(id int64) func(context.Context, *http.Client) {
	return func(ctx context.Context, c *http.Client) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, *baseURL+"/pets/"+itoa(id), nil)
		resp, err := c.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}
}

func createPetTask(name string) func(context.Context, *http.Client) {
	return func(ctx context.Context, c *http.Client) {
		body, _ := json.Marshal(map[string]any{"name": name})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, *baseURL+"/pets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}
}

func itoa(v int64) string {
	// small, avoids strconv import in this tiny tool
	if v == 0 {
		return "0"
	}

	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}

	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + (v % 10))
		v /= 10
	}
	return sign + string(b[i:])
}
