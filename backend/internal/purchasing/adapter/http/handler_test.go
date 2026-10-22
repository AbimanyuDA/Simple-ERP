package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authapp "github.com/simple-erp/backend/internal/auth/application"
	authdomain "github.com/simple-erp/backend/internal/auth/domain"
)

func TestPurchasingMakerCheckerRejectsWrongRole(t *testing.T) {
	h := Handler{}
	tests := []struct {
		name, role string
		handler    http.HandlerFunc
	}{
		{"staff cannot issue purchase order", authdomain.RoleStaff, h.issue},
		{"staff cannot cancel purchase order", authdomain.RoleStaff, h.cancelOrder},
		{"staff cannot post goods receipt", authdomain.RoleStaff, h.postReceipt},
		{"supervisor cannot create purchase request", authdomain.RoleSupervisor, h.create},
		{"supervisor cannot create goods receipt", authdomain.RoleSupervisor, h.receipt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req = req.WithContext(authapp.WithPrincipal(req.Context(), authapp.Principal{UserID: "actor", Role: tt.role}))
			response := httptest.NewRecorder()
			tt.handler(response, req)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status=%d, want 403", response.Code)
			}
		})
	}
}
