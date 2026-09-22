package idempotency

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	authapp "github.com/simple-erp/backend/internal/auth/application"
)

const maxRequestBody = 1 << 20

type Record struct {
	RequestHash string
	Status      int
	Body        []byte
}

type Store interface {
	Claim(context.Context, string, string, string) (Record, bool, error)
	Complete(context.Context, string, string, int, []byte) error
	Release(context.Context, string, string) error
}

func Middleware(store Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			principal, err := authapp.PrincipalFrom(r.Context())
			if err != nil {
				writeError(w, http.StatusUnauthorized, "Autentikasi diperlukan.")
				return
			}
			key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if key == "" || len(key) > 200 {
				writeError(w, http.StatusBadRequest, "Idempotency-Key wajib diisi.")
				return
			}
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
			if err != nil {
				writeError(w, http.StatusBadRequest, "Data tidak valid.")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			digest := sha256.Sum256(append([]byte(r.Method+"\n"+r.URL.Path+"\n"), body...))
			hash := hex.EncodeToString(digest[:])

			record, claimed, err := store.Claim(r.Context(), principal.UserID, key, hash)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses.")
				return
			}
			if !claimed {
				if record.RequestHash != hash {
					writeError(w, http.StatusConflict, "Idempotency-Key telah digunakan untuk payload berbeda.")
					return
				}
				if record.Status == 0 {
					writeError(w, http.StatusConflict, "Permintaan dengan key ini sedang diproses.")
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Idempotency-Replayed", "true")
				w.WriteHeader(record.Status)
				_, _ = w.Write(record.Body)
				return
			}

			captured := newCapture()
			next.ServeHTTP(captured, r)
			if captured.status >= http.StatusInternalServerError {
				_ = store.Release(r.Context(), principal.UserID, key)
			} else if err := store.Complete(r.Context(), principal.UserID, key, captured.status, captured.body.Bytes()); err != nil {
				writeError(w, http.StatusInternalServerError, "Permintaan selesai tetapi respons tidak dapat disimpan.")
				return
			}
			for name, values := range captured.header {
				for _, value := range values {
					w.Header().Add(name, value)
				}
			}
			w.WriteHeader(captured.status)
			_, _ = w.Write(captured.body.Bytes())
		})
	}
}

type capture struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newCapture() *capture                        { return &capture{header: make(http.Header), status: http.StatusOK} }
func (c *capture) Header() http.Header            { return c.header }
func (c *capture) WriteHeader(status int)         { c.status = status }
func (c *capture) Write(body []byte) (int, error) { return c.body.Write(body) }

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}
