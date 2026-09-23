package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
)

type authFake struct{}

func (authFake) Authenticate(context.Context, string) (Principal, error) {
	return Principal{UserID: "u1", Role: "ADMIN", CSRFHash: "hash"}, nil
}
func (authFake) VerifyCSRF(string, string) bool { return true }
func TestProtectedRouteRejectsMissingSession(t *testing.T) {
	m := NewMiddleware(authFake{}, "https://erp.example.test")
	rec := httptest.NewRecorder()
	m.RequireAuth(stdhttp.HandlerFunc(func(stdhttp.ResponseWriter, *stdhttp.Request) { t.Fatal("called") })).ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	if rec.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestAuthenticatedTrafficIsRateLimitedPerUser(t *testing.T) {
	m := NewMiddleware(authFake{}, "https://erp.example.test")
	handler := m.RequireAuth(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) { w.WriteHeader(stdhttp.StatusOK) }))
	for attempt := 1; attempt <= 121; attempt++ {
		req := httptest.NewRequest(stdhttp.MethodGet, "/dashboard/summary", nil)
		req.AddCookie(&stdhttp.Cookie{Name: "__Host-erp_access", Value: "token"})
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if attempt <= 120 && recorder.Code != stdhttp.StatusOK {
			t.Fatalf("attempt %d status=%d", attempt, recorder.Code)
		}
		if attempt == 121 && (recorder.Code != stdhttp.StatusTooManyRequests || recorder.Header().Get("Retry-After") == "") {
			t.Fatalf("limit status=%d retry=%q", recorder.Code, recorder.Header().Get("Retry-After"))
		}
	}
}
