package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"openapi-mock/pkg/ctxkeys"
	"openapi-mock/pkg/metrics"
	"openapi-mock/pkg/recorder"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func Recording(rec *recorder.Recorder, m *metrics.Metrics, enableLogging bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b := make([]byte, 8)
			rand.Read(b)
			reqID := fmt.Sprintf("%x", b)
			start := time.Now()

			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			rw := &responseWriter{ResponseWriter: w, statusCode: 200}
			ctx := context.WithValue(r.Context(), ctxkeys.RequestID{}, reqID)
			r = r.WithContext(ctx)

			defer func() {
				if err := recover(); err != nil {
					rec.Record(recorder.CallRecord{
						RequestID:  reqID,
						Method:     r.Method + " " + r.URL.Path,
						Timestamp:  start,
						Request:    string(bodyBytes),
						Panic:      fmt.Sprintf("%v", err),
						DurationMs: time.Since(start).Milliseconds(),
					})
					if m != nil {
						m.RecordHTTPRequest(r.Method, r.URL.Path, time.Since(start).Milliseconds(), "panic")
					}
					http.Error(w, "Internal Server Error", 500)
				}
			}()

			if enableLogging {
				log.Printf("[req_id=%s] --> %s %s", reqID, r.Method, r.URL.Path)
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			status := "success"
			if rw.statusCode >= 400 && rw.statusCode < 500 {
				status = "client_error"
			} else if rw.statusCode >= 500 {
				status = "server_error"
			}

			rec.Record(recorder.CallRecord{
				RequestID:  reqID,
				Method:     r.Method + " " + r.URL.Path,
				Timestamp:  start,
				Request:    string(bodyBytes),
				Response:   rw.body.String(),
				DurationMs: duration.Milliseconds(),
			})

			if m != nil {
				m.RecordHTTPRequest(r.Method, r.URL.Path, duration.Milliseconds(), status)
			}

			if enableLogging {
				log.Printf("[req_id=%s] <-- %d (%dms)", reqID, rw.statusCode, duration.Milliseconds())
			}
		})
	}
}
