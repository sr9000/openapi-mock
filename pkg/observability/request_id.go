package observability

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

const DefaultRequestIDResponseHeader = "X-Request-ID"

func NormalizeHeaderList(headersCSV string, defaults []string) []string {
	if strings.TrimSpace(headersCSV) == "" {
		return defaults
	}
	parts := strings.Split(headersCSV, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		h := strings.TrimSpace(p)
		if h != "" {
			result = append(result, h)
		}
	}
	if len(result) == 0 {
		return defaults
	}
	return result
}

func ResolveRequestID(getHeader func(string) string, allowedHeaders []string) string {
	for _, h := range allowedHeaders {
		v := strings.TrimSpace(getHeader(h))
		if v != "" {
			return v
		}
	}
	return GenerateRequestID()
}

func GenerateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		now := time.Now().UnixNano()
		for i := 0; i < len(b); i++ {
			b[i] = byte(now >> (8 * i))
		}
	}
	return fmt.Sprintf("%x", b)
}

func TraceIDFromTraceparent(v string) string {
	parts := strings.Split(strings.TrimSpace(v), "-")
	if len(parts) != 4 {
		return ""
	}
	if len(parts[1]) != 32 {
		return ""
	}
	return strings.ToLower(parts[1])
}
