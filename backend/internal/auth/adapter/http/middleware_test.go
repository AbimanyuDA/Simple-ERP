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
