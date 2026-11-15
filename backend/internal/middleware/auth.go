package middleware

import (
	"context"
	"factorymatch/internal/auth"
	"factorymatch/internal/models"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

type contextKey string

const userKey contextKey = "user"

// UserFromContext returns the authenticated user stored by Auth, if any.
func UserFromContext(r *http.Request) (models.User, bool) {
	u, ok := r.Context().Value(userKey).(models.User)
	return u, ok
}

// Auth verifies the Bearer token and loads the user into the request
// context. Requests without a valid token get 401.
func Auth(db *gorm.DB, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if db == nil {
				http.Error(w, `{"error":"database not configured — set DATABASE_URL"}`, http.StatusServiceUnavailable)
				return
			}
			token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if token == "" {
				http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
				return
			}
			claims, err := auth.Verify(token, secret)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			var user models.User
			if err := db.First(&user, claims.UserID).Error; err != nil {
				http.Error(w, `{"error":"user not found"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole allows only users whose role is in the list.
func RequireRole(roles ...models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r)
			if !ok {
				http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
				return
			}
			if len(roles) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		})
	}
}
