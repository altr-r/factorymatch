package handlers

import (
	"encoding/json"
	"net/http"

	"gorm.io/gorm"

	"factorymatch/internal/config"
	"factorymatch/internal/middleware"
	"factorymatch/internal/models"
)

// Handler carries the shared dependencies for all endpoints.
type Handler struct {
	DB        *gorm.DB // nil when DATABASE_URL is not set
	JWTSecret string
}

// RegisterRoutes mounts every route on mux.
func RegisterRoutes(mux *http.ServeMux, db *gorm.DB, cfg config.Config) {
	h := &Handler{DB: db, JWTSecret: cfg.JWTSecret}

	// Middleware helpers.
	authed := func(roles ...models.Role) func(http.Handler) http.Handler {
		auth := middleware.Auth(db, cfg.JWTSecret)
		return func(next http.Handler) http.Handler {
			return auth(middleware.RequireRole(roles...)(next))
		}
	}
	chain := func(handler http.HandlerFunc, mws ...func(http.Handler) http.Handler) http.Handler {
		var wrapped http.Handler = handler
		for i := len(mws) - 1; i >= 0; i-- {
			wrapped = mws[i](wrapped)
		}
		return wrapped
	}

	// Health — works without a database.
	mux.Handle("GET /healthz", chain(h.health, middleware.Logger, middleware.CORS(cfg.FrontendOrigin)))

	// Uploaded workshop photos (1.2) are served from ./uploads.
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// C.1 Registration and Login.
	mux.Handle("POST /api/auth/register", chain(h.register, middleware.Logger, middleware.CORS(cfg.FrontendOrigin)))
	mux.Handle("POST /api/auth/login", chain(h.login, middleware.Logger, middleware.CORS(cfg.FrontendOrigin)))
	mux.Handle("POST /api/auth/logout", chain(h.logout, middleware.Logger, middleware.CORS(cfg.FrontendOrigin)))
	mux.Handle("GET /api/auth/me", chain(h.me, middleware.Logger, middleware.CORS(cfg.FrontendOrigin), authed()))
	mux.Handle("GET /api/auth/me/status", chain(h.meStatus, middleware.Logger, middleware.CORS(cfg.FrontendOrigin), authed()))

	// 1.2 Manufacturer Workshop Profile.
	mux.Handle("GET /api/manufacturer/workshop", chain(h.getMyWorkshop, middleware.Logger, middleware.CORS(cfg.FrontendOrigin), authed(models.RoleManufacturer)))
	mux.Handle("POST /api/workshops", chain(h.createWorkshop, middleware.Logger, middleware.CORS(cfg.FrontendOrigin), authed(models.RoleManufacturer)))
	mux.Handle("GET /api/workshops/{id}", chain(h.getWorkshop, middleware.Logger, middleware.CORS(cfg.FrontendOrigin), authed()))
	mux.Handle("PUT /api/workshops/{id}", chain(h.updateWorkshop, middleware.Logger, middleware.CORS(cfg.FrontendOrigin), authed(models.RoleManufacturer)))
	mux.Handle("GET /api/workshops/{id}/public", chain(h.getPublicWorkshop, middleware.Logger, middleware.CORS(cfg.FrontendOrigin)))
	mux.Handle("POST /api/workshops/{id}/photos", chain(h.uploadWorkshopPhoto, middleware.Logger, middleware.CORS(cfg.FrontendOrigin), authed(models.RoleManufacturer)))
	mux.Handle("GET /api/workshops/{id}/photos", chain(h.listWorkshopPhotos, middleware.Logger, middleware.CORS(cfg.FrontendOrigin)))
	mux.Handle("DELETE /api/workshops/{id}/photos/{photoId}", chain(h.deleteWorkshopPhoto, middleware.Logger, middleware.CORS(cfg.FrontendOrigin), authed(models.RoleManufacturer)))

}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// requireDB answers 503 when the server runs without DATABASE_URL.
func (h *Handler) requireDB(w http.ResponseWriter) bool {
	if h.DB == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "database not configured — set DATABASE_URL",
		})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// currentUser returns the authenticated user or answers 401.
func currentUser(w http.ResponseWriter, r *http.Request) (models.User, bool) {
	user, ok := middleware.UserFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return user, false
	}
	return user, true
}
