package mm

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"sync"
)

type valuesKey struct{}

// WithValues stores request-scoped values in context.
func WithValues(ctx context.Context, values map[string]any) context.Context {
	if len(values) == 0 {
		return context.WithValue(ctx, valuesKey{}, map[string]any{})
	}
	return context.WithValue(ctx, valuesKey{}, cloneMap(values))
}

// FromCtx returns a value by name or nil if it is absent.
func FromCtx(ctx context.Context, name string) any {
	v, ok := Lookup(ctx, name)
	if !ok {
		return nil
	}
	return v
}

// Lookup returns a value by name and whether it was found.
func Lookup(ctx context.Context, name string) (any, bool) {
	values, _ := ctx.Value(valuesKey{}).(map[string]any)
	if values == nil {
		return nil, false
	}
	v, ok := values[name]
	if !ok {
		return nil, false
	}
	return cloneValue(v), true
}

// Store keeps request-id keyed context values.
type Store struct {
	mu   sync.RWMutex
	data map[string]map[string]any
}

func NewStore() *Store {
	return &Store{data: make(map[string]map[string]any)}
}

func (s *Store) Get(requestID string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneMap(s.data[requestID])
}

func (s *Store) Replace(requestID string, values map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(values) == 0 {
		delete(s.data, requestID)
		return
	}
	s.data[requestID] = cloneMap(values)
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]map[string]any)
}

// DecodeObject decodes JSON object with numeric normalization.
func DecodeObject(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	return normalizeMap(payload), nil
}

func normalizeMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = normalizeValue(v)
	}
	return out
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return normalizeMap(t)
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = normalizeValue(t[i])
		}
		return out
	case json.Number:
		s := t.String()
		if strings.ContainsAny(s, ".eE") {
			f, err := t.Float64()
			if err != nil {
				return s
			}
			return f
		}
		i64, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			f, ferr := t.Float64()
			if ferr != nil {
				return s
			}
			return f
		}
		if i64 >= int64(math.MinInt) && i64 <= int64(math.MaxInt) {
			return int(i64)
		}
		return i64
	default:
		return t
	}
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return cloneMap(t)
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = cloneValue(t[i])
		}
		return out
	default:
		return t
	}
}
