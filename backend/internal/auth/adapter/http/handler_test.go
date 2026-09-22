package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/simple-erp/backend/internal/auth/domain"
)

type loginFake struct{}

func (loginFake) Login(context.Context, string, string, string) (domain.PlainTokens, error) {
	return domain.PlainTokens{Access: "access", Refresh: "refresh", CSRF: "csrf"}, nil
}

func TestCAPAUTH001LoginSetsHostOnlySecureCookies(t *testing.T) {
	h := NewHandler(loginFake{})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"admin@example.test","password":"Kuat-sekali-2026!"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 3 {
		t.Fatalf("cookies=%d", len(cookies))
	}
	for _, c := range cookies {
		if c.Name == "erp_csrf" {
			if c.HttpOnly || !c.Secure {
				t.Fatalf("invalid csrf cookie: %#v", c)
			}
			continue
		}
		if !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.Domain != "" {
			t.Fatalf("unsafe cookie: %#v", c)
		}
		if c.Expires.Before(time.Now()) {
			t.Fatal("cookie already expired")
		}
	}
}
