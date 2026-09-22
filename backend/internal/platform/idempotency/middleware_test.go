package idempotency

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authapp "github.com/simple-erp/backend/internal/auth/application"
)

type memoryStore struct {
	records map[string]Record
}

func (s *memoryStore) Claim(_ context.Context, actor, key, hash string) (Record, bool, error) {
	id := actor + ":" + key
	if existing, ok := s.records[id]; ok {
		return existing, false, nil
	}
	record := Record{RequestHash: hash}
	s.records[id] = record
	return record, true, nil
}

func (s *memoryStore) Complete(_ context.Context, actor, key string, status int, body []byte) error {
	record := s.records[actor+":"+key]
	record.Status = status
	record.Body = append([]byte(nil), body...)
	s.records[actor+":"+key] = record
	return nil
}

func (s *memoryStore) Release(_ context.Context, actor, key string) error {
	delete(s.records, actor+":"+key)
	return nil
}

func TestMutationReplayReturnsOriginalResponseWithoutRunningHandlerTwice(t *testing.T) {
	store := &memoryStore{records: map[string]Record{}}
	calls := 0
	handler := Middleware(store)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"one"}`))
	}))

	invoke := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/documents", strings.NewReader(body))
		req.Header.Set("Idempotency-Key", "stable-key")
		req = req.WithContext(authapp.WithPrincipal(req.Context(), authapp.Principal{UserID: "actor-1"}))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder
	}

	first := invoke(`{"value":1}`)
	second := invoke(`{"value":1}`)
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated || second.Body.String() != first.Body.String() {
		t.Fatalf("first=%d/%s second=%d/%s", first.Code, first.Body.String(), second.Code, second.Body.String())
	}
	if calls != 1 {
		t.Fatalf("handler calls=%d", calls)
	}
}

func TestMutationRejectsMissingOrReusedKeyWithDifferentPayload(t *testing.T) {
	store := &memoryStore{records: map[string]Record{}}
	handler := Middleware(store)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	request := func(key, body string) int {
		req := httptest.NewRequest(http.MethodPost, "/documents", strings.NewReader(body))
		req.Header.Set("Idempotency-Key", key)
		req = req.WithContext(authapp.WithPrincipal(req.Context(), authapp.Principal{UserID: "actor-1"}))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder.Code
	}
	if got := request("", `{}`); got != http.StatusBadRequest {
		t.Fatalf("missing key status=%d", got)
	}
	if got := request("stable-key", `{"value":1}`); got != http.StatusOK {
		t.Fatalf("initial status=%d", got)
	}
	if got := request("stable-key", `{"value":2}`); got != http.StatusConflict {
		t.Fatalf("mismatched replay status=%d", got)
	}
}
