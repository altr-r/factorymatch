package handlers

import (
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"factorymatch/internal/auth"
	"factorymatch/internal/models"
)

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"` // buyer or manufacturer only
}

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// register creates a buyer or manufacturer account. Admin accounts are
// created by FactoryMatch, never self-registered.
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	var req registerRequest
	if !readJSON(w, r, &req) {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "name, email and a password of at least 8 characters are required")
		return
	}
	role := models.Role(req.Role)
	if role != models.RoleBuyer && role != models.RoleManufacturer {
		writeError(w, http.StatusBadRequest, "role must be buyer or manufacturer")
		return
	}
	var existing models.User
	if err := h.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	user := models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         role,
		Status:       models.StatusActive,
	}
	if err := h.DB.Create(&user).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not create account")
		return
	}
	token, err := auth.Issue(user.ID, string(user.Role), h.JWTSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// login signs a user in and returns a token.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	var req loginRequest
	if !readJSON(w, r, &req) {
		return
	}
	var user models.User
	if err := h.DB.Where("email = ?", strings.ToLower(strings.TrimSpace(req.Email))).First(&user).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if user.Status == models.StatusSuspended {
		writeError(w, http.StatusForbidden, "account suspended")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := auth.Issue(user.ID, string(user.Role), h.JWTSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

// logout is a no-op for stateless JWTs: the client drops the token.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed out — discard the token client-side"})
}

// me returns the signed-in user's account.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// meStatus returns the account's registration details and current status.
func (h *Handler) meStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"status":     user.Status,
		"created_at": user.CreatedAt,
	})
}
