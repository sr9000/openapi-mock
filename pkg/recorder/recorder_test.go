package recorder

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
	if len(r.GetRecords()) != 0 {
		t.Error("New recorder should have no records")
	}
}

func TestRecord(t *testing.T) {
	r := New()

	record := CallRecord{
		RequestID:  "test-id-1",
		Method:     "GET /pets",
		StatusCode: 200,
		Path:       "/pets",
		Query:      "limit=10",
		Timestamp:  time.Now(),
		Request:    json.RawMessage(`{"message":"hello"}`),
		Response:   json.RawMessage(`{"message":"world"}`),
		DurationMs: 100,
	}

	r.Record(record)

	records := r.GetRecords()
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	if records[0].RequestID != "test-id-1" {
		t.Errorf("Expected request_id 'test-id-1', got '%s'", records[0].RequestID)
	}
	if records[0].Method != "GET /pets" {
		t.Errorf("Expected method 'GET /pets', got '%s'", records[0].Method)
	}
	if records[0].StatusCode != 200 {
		t.Errorf("Expected status_code 200, got %d", records[0].StatusCode)
	}
	if records[0].Path != "/pets" {
		t.Errorf("Expected path '/pets', got '%s'", records[0].Path)
	}
	if records[0].Query != "limit=10" {
		t.Errorf("Expected query 'limit=10', got '%s'", records[0].Query)
	}
}

func TestRecordMultiple(t *testing.T) {
	r := New()

	for i := 0; i < 5; i++ {
		r.Record(CallRecord{
			RequestID: "test-id",
			Method:    "GET /a",
			Path:      "/a",
			Timestamp: time.Now(),
		})
	}

	records := r.GetRecords()
	if len(records) != 5 {
		t.Fatalf("Expected 5 records, got %d", len(records))
	}
}

func TestClear(t *testing.T) {
	r := New()

	r.Record(CallRecord{
		RequestID: "test-id",
		Method:    "GET /a",
		Path:      "/a",
		Timestamp: time.Now(),
	})

	if len(r.GetRecords()) != 1 {
		t.Fatal("Expected 1 record before clear")
	}

	r.Clear()

	if len(r.GetRecords()) != 0 {
		t.Error("Expected 0 records after clear")
	}
}

func TestToJSON(t *testing.T) {
	r := New()

	timestamp := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	r.Record(CallRecord{
		RequestID:  "test-id",
		Method:     "GET /a",
		StatusCode: 200,
		Path:       "/a",
		Timestamp:  timestamp,
		Request:    json.RawMessage(`{"message":"hello"}`),
		Response:   json.RawMessage(`{"message":"world"}`),
		DurationMs: 50,
	})

	data, err := r.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	var records []CallRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	if records[0].RequestID != "test-id" {
		t.Errorf("Expected request_id 'test-id', got '%s'", records[0].RequestID)
	}
	if records[0].StatusCode != 200 {
		t.Errorf("Expected status_code 200, got %d", records[0].StatusCode)
	}
}

func TestRecordWithError(t *testing.T) {
	r := New()

	r.Record(CallRecord{
		RequestID:  "test-id",
		Method:     "GET /a",
		Path:       "/a",
		Timestamp:  time.Now(),
		Error:      "something went wrong",
		DurationMs: 10,
	})

	records := r.GetRecords()
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	if records[0].Error != "something went wrong" {
		t.Errorf("Expected error 'something went wrong', got '%s'", records[0].Error)
	}
}

func TestRecordWithPanic(t *testing.T) {
	r := New()

	r.Record(CallRecord{
		RequestID:  "test-id",
		Method:     "GET /a",
		Path:       "/a",
		Timestamp:  time.Now(),
		Panic:      "runtime error: index out of range",
		DurationMs: 5,
	})

	records := r.GetRecords()
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	if records[0].Panic != "runtime error: index out of range" {
		t.Errorf("Expected panic 'runtime error: index out of range', got '%s'", records[0].Panic)
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := New()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			r.Record(CallRecord{
				RequestID: "test-id",
				Method:    "GET /a",
				Path:      "/a",
				Timestamp: time.Now(),
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.GetRecords()
		}()
	}

	// Concurrent clears
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Clear()
		}()
	}

	wg.Wait()

	// Should not panic and should complete without data races
}

func TestGetRecordsReturnsCopy(t *testing.T) {
	r := New()

	r.Record(CallRecord{
		RequestID: "test-id-1",
		Method:    "GET /a",
		Path:      "/a",
		Timestamp: time.Now(),
	})

	records1 := r.GetRecords()
	records1[0].RequestID = "modified"

	records2 := r.GetRecords()
	if records2[0].RequestID != "test-id-1" {
		t.Error("GetRecords should return a copy, not the original slice")
	}
}

func TestGetRecordsByRequestID(t *testing.T) {
	r := New()
	r.Record(CallRecord{RequestID: "id-1", Method: "GET /a", Path: "/a", Timestamp: time.Now()})
	r.Record(CallRecord{RequestID: "id-2", Method: "GET /b", Path: "/b", Timestamp: time.Now()})
	r.Record(CallRecord{RequestID: "id-1", Method: "GET /c", Path: "/c", Timestamp: time.Now()})

	records := r.GetRecordsByRequestID("id-1")
	if len(records) != 2 {
		t.Fatalf("Expected 2 records for id-1, got %d", len(records))
	}

	for _, record := range records {
		if record.RequestID != "id-1" {
			t.Fatalf("Expected only id-1 records, got %q", record.RequestID)
		}
	}
}

func TestCallRecordJSONRoundTrip(t *testing.T) {
	original := CallRecord{
		RequestID:  "round-trip",
		Method:     "POST /echo",
		StatusCode: 201,
		Path:       "/echo",
		Query:      "q=test",
		Timestamp:  time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC),
		Request:    json.RawMessage(`{"message":"hi"}`),
		Response:   json.RawMessage(`{"message":"hello"}`),
		DurationMs: 42,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded CallRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.StatusCode != 201 {
		t.Errorf("expected status_code 201, got %d", decoded.StatusCode)
	}
	if decoded.Path != "/echo" {
		t.Errorf("expected path /echo, got %q", decoded.Path)
	}
	if decoded.Query != "q=test" {
		t.Errorf("expected query q=test, got %q", decoded.Query)
	}
	if string(decoded.Request) != `{"message":"hi"}` {
		t.Errorf("expected request body, got %q", string(decoded.Request))
	}
	if string(decoded.Response) != `{"message":"hello"}` {
		t.Errorf("expected response body, got %q", string(decoded.Response))
	}
}
