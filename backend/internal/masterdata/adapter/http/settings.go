package http

import (
	"encoding/json"
	"github.com/go-chi/chi/v5/middleware"
	authapp "github.com/simple-erp/backend/internal/auth/application"
	mdapp "github.com/simple-erp/backend/internal/masterdata/application"
	"github.com/simple-erp/backend/internal/masterdata/domain"
	"net/http"
)

type SettingsHandler struct{ Service mdapp.SettingsService }

func (h SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		v, e := h.Service.Get(r.Context())
		if e != nil {
			respond(w, 500, map[string]string{"message": "Pengaturan tidak tersedia."})
			return
		}
		respond(w, 200, v)
	case http.MethodPatch:
		var v domain.OrganizationSettings
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			respond(w, 400, map[string]string{"message": "Data tidak valid."})
			return
		}
		p, _ := authapp.PrincipalFrom(r.Context())
		if e := h.Service.Update(r.Context(), v, p.UserID, middleware.GetReqID(r.Context())); e != nil {
			respond(w, 409, map[string]string{"message": e.Error()})
			return
		}
		respond(w, 200, v)
	default:
		w.WriteHeader(405)
	}
}
