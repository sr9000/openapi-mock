package middleware

import (
	"net/http"
	"strings"
	"sync"

	"openapi-mock/pkg/observability"
)

type OperationResolver map[string]string

func NewOperationResolver(entries map[string]string) OperationResolver {
	resolver := make(OperationResolver, len(entries))
	for k, v := range entries {
		resolver[k] = v
	}
	return resolver
}

func (r OperationResolver) Resolve(method, route string) string {
	if len(r) == 0 {
		return ""
	}
	if operation := r[resolverKey(method, route)]; operation != "" {
		return operation
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	route = strings.TrimSpace(route)
	for key, operation := range r {
		parts := strings.SplitN(key, " ", 2)
		if len(parts) != 2 || parts[0] != method {
			continue
		}
		if routeMatches(parts[1], route) {
			return operation
		}
	}
	return ""
}

var (
	operationResolverMu sync.RWMutex
	globalResolver      OperationResolver
)

func SetOperationResolver(resolver OperationResolver) {
	operationResolverMu.Lock()
	defer operationResolverMu.Unlock()
	globalResolver = resolver
}

func GlobalOperationResolver() OperationResolver {
	operationResolverMu.RLock()
	defer operationResolverMu.RUnlock()
	return globalResolver
}

func resolveOperationLabel(r *http.Request, resolver OperationResolver) string {
	if operation := observability.Operation(r.Context()); operation != "" {
		return operation
	}
	if resolver == nil {
		resolver = GlobalOperationResolver()
	}
	if resolver != nil {
		if operation := resolver.Resolve(r.Method, routeTemplateFromRequest(r)); operation != "" {
			return operation
		}
	}
	return "unknown"
}

func resolverKey(method, route string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(route)
}

func routeMatches(template, actual string) bool {
	tplParts := strings.Split(strings.Trim(template, "/"), "/")
	actParts := strings.Split(strings.Trim(actual, "/"), "/")
	if len(tplParts) != len(actParts) {
		return false
	}
	for i := range tplParts {
		if strings.HasPrefix(tplParts[i], "{") && strings.HasSuffix(tplParts[i], "}") {
			continue
		}
		if tplParts[i] != actParts[i] {
			return false
		}
	}
	return true
}
