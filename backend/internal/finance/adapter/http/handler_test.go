package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authapp "github.com/simple-erp/backend/internal/auth/application"
	authdomain "github.com/simple-erp/backend/internal/auth/domain"
)

func TestSupplierMakerCheckerRejectsWrongRole(t *testing.T) {
	h := Handler{}
	tests := []struct {
		name, role string
		handler    http.HandlerFunc
	}{
		{"supervisor cannot create supplier invoice", authdomain.RoleSupervisor, h.createSupplierInvoice},
		{"supervisor cannot resolve supplier invoice", authdomain.RoleSupervisor, h.resolveSupplierInvoice},
		{"supervisor cannot create supplier payment", authdomain.RoleSupervisor, h.createSupplierPayment},
		{"staff cannot match supplier invoice", authdomain.RoleStaff, h.matchSupplierInvoice},
		{"staff cannot void supplier invoice", authdomain.RoleStaff, h.voidSupplierInvoice},
		{"staff cannot post supplier payment", authdomain.RoleStaff, h.postSupplierPayment},
		{"staff cannot void supplier payment", authdomain.RoleStaff, h.voidSupplierPayment},
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

func TestSalesMakerCheckerRejectsWrongRole(t *testing.T) {
	h := Handler{}
	tests := []struct {
		name, role string
		handler    http.HandlerFunc
	}{
		{"supervisor cannot create sales invoice", authdomain.RoleSupervisor, h.createSalesInvoice},
		{"supervisor cannot create customer payment", authdomain.RoleSupervisor, h.createCustomerPayment},
		{"staff cannot post sales invoice", authdomain.RoleStaff, h.postSalesInvoice},
		{"staff cannot void sales invoice", authdomain.RoleStaff, h.voidSalesInvoice},
		{"staff cannot post customer payment", authdomain.RoleStaff, h.postCustomerPayment},
		{"staff cannot void customer payment", authdomain.RoleStaff, h.voidCustomerPayment},
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
