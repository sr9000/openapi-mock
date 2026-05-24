package echo

import (
	"context"
	"testing"

	gen "openapi-mock/internal/generated/echo"
	"openapi-mock/pkg/observability"
)

func TestEchoHandlers_Echo(t *testing.T) {
	h := NewEchoHandlers(false)
	msg := "hello"
	resp, err := h.Echo(context.Background(), gen.EchoRequestObject{JSONBody: &gen.EchoJSONRequestBody{Message: &msg}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	jsonResp, ok := resp.(gen.Echo200JSONResponse)
	if !ok {
		t.Fatalf("expected Echo200JSONResponse, got %T", resp)
	}
	if jsonResp.Echo != "hello" {
		t.Fatalf("expected echoed message 'hello', got %q", jsonResp.Echo)
	}
}

func TestEchoHandlers_EchoPath(t *testing.T) {
	h := NewEchoHandlers(false)
	resp, err := h.EchoPath(context.Background(), gen.EchoPathRequestObject{Message: "path-msg"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	jsonResp, ok := resp.(gen.EchoPath200JSONResponse)
	if !ok {
		t.Fatalf("expected EchoPath200JSONResponse, got %T", resp)
	}
	if jsonResp.Echo != "path-msg" {
		t.Fatalf("expected echoed path message 'path-msg', got %q", jsonResp.Echo)
	}
}

func TestEchoHandlers_EchoHeaders(t *testing.T) {
	h := NewEchoHandlers(false)
	ctx := observability.WithTraceID(observability.WithRequestID(context.Background(), "req-1"), "trace-1")
	resp, err := h.EchoHeaders(ctx, gen.EchoHeadersRequestObject{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	jsonResp, ok := resp.(gen.EchoHeaders200JSONResponse)
	if !ok {
		t.Fatalf("expected EchoHeaders200JSONResponse, got %T", resp)
	}
	if jsonResp.Headers == nil || (*jsonResp.Headers)["X-Request-ID"] != "req-1" {
		t.Fatalf("expected X-Request-ID to be echoed")
	}
}
