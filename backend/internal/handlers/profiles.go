package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"factorymatch/internal/models"
)

type workshopRequest struct {
	Name               string `json:"name"`
	Address            string `json:"address"`
	ContactDetails     string `json:"contact_details"`
	OperatingLocations string `json:"operating_locations"`
	Description        string `json:"description"`
}

// loadWorkshop loads the workshop named by the {id} path value.
// Bad id → 400, missing workshop → 404.
func (h *Handler) loadWorkshop(w http.ResponseWriter, r *http.Request) (models.Workshop, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workshop id")
		return models.Workshop{}, false
	}
	var ws models.Workshop
	if err := h.DB.First(&ws, id).Error; err != nil {
		writeError(w, http.StatusNotFound, "workshop not found")
		return models.Workshop{}, false
	}
	return ws, true
}

// createWorkshop creates the signed-in manufacturer's workshop.
// Manufacturer role only (enforced on the route). Name and address are
// required. New workshops start unverified, verification pending —
// verification itself is C.2's job, not this endpoint's.
func (h *Handler) createWorkshop(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	var req workshopRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.Name == "" || req.Address == "" {
		writeError(w, http.StatusBadRequest, "name and address are required")
		return
	}
	ws := models.Workshop{
		UserID:             user.ID,
		Name:               req.Name,
		Address:            req.Address,
		ContactDetails:     req.ContactDetails,
		OperatingLocations: req.OperatingLocations,
		Description:        req.Description,
		VerificationStatus: "pending",
	}
	if err := h.DB.Create(&ws).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not create workshop")
		return
	}
	writeJSON(w, http.StatusCreated, ws)
}

// getMyWorkshop returns the signed-in manufacturer's own workshop
// (the earliest one they created). 404 when they have not created one yet.
func (h *Handler) getMyWorkshop(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	var ws models.Workshop
	if err := h.DB.Where("user_id = ?", user.ID).Order("id").First(&ws).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "no workshop yet")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

// getWorkshop returns the full profile to its owner or an admin; anyone
// else gets the public view, and only when the workshop is verified.
func (h *Handler) getWorkshop(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	ws, ok := h.loadWorkshop(w, r)
	if !ok {
		return
	}
	if ws.UserID == user.ID || user.Role == models.RoleAdmin {
		writeJSON(w, http.StatusOK, ws)
		return
	}
	if !ws.Verified {
		writeError(w, http.StatusNotFound, "workshop not found")
		return
	}
	h.writePublicWorkshop(w, ws)
}

// updateWorkshop edits a workshop. Owner only: anyone else gets 403.
// Only the five profile fields change — Verified, VerificationStatus and
// UserID are never touched here.
func (h *Handler) updateWorkshop(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	ws, ok := h.loadWorkshop(w, r)
	if !ok {
		return
	}
	if ws.UserID != user.ID {
		writeError(w, http.StatusForbidden, "not your workshop")
		return
	}
	var req workshopRequest
	if !readJSON(w, r, &req) {
		return
	}
	ws.Name = req.Name
	ws.Address = req.Address
	ws.ContactDetails = req.ContactDetails
	ws.OperatingLocations = req.OperatingLocations
	ws.Description = req.Description
	if err := h.DB.Save(&ws).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not update workshop")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

type publicWorkshopResponse struct {
	models.Workshop
	Photos       []models.WorkshopPhoto `json:"photos"`
	Capabilities []models.Capability    `json:"capabilities"`
}

// getPublicWorkshop is the buyer-facing view: public details of a verified
// workshop only (404 otherwise). Verification documents are never part of
// this model, so they can never leak through this endpoint.
func (h *Handler) getPublicWorkshop(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	ws, ok := h.loadWorkshop(w, r)
	if !ok {
		return
	}
	if !ws.Verified {
		writeError(w, http.StatusNotFound, "workshop not found")
		return
	}
	h.writePublicWorkshop(w, ws)
}

func (h *Handler) writePublicWorkshop(w http.ResponseWriter, ws models.Workshop) {
	photos := []models.WorkshopPhoto{}
	caps := []models.Capability{}
	_ = h.DB.Where("workshop_id = ?", ws.ID).Find(&photos).Error
	_ = h.DB.Where("workshop_id = ?", ws.ID).Find(&caps).Error
	writeJSON(w, http.StatusOK, publicWorkshopResponse{
		Workshop:     ws,
		Photos:       photos,
		Capabilities: caps,
	})
}

// --- Previous-work photos ---

var allowedPhotoExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}

// uploadWorkshopPhoto stores one previous-work photo (multipart form,
// field "photo", optional "caption"). 10 MB max, jpg/png/webp only.
// Files go to ./uploads/ and are served at /uploads/ — run the server
// from backend/ so the relative path resolves.
func (h *Handler) uploadWorkshopPhoto(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	ws, ok := h.loadWorkshop(w, r)
	if !ok {
		return
	}
	if ws.UserID != user.ID {
		writeError(w, http.StatusForbidden, "not your workshop")
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB
		writeError(w, http.StatusBadRequest, "photo too large (max 10 MB)")
		return
	}
	file, header, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing photo file")
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedPhotoExts[ext] {
		writeError(w, http.StatusBadRequest, "photo must be jpg, png or webp")
		return
	}
	if err := os.MkdirAll("uploads", 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "could not store photo")
		return
	}
	name := fmt.Sprintf("%d_%d%s", ws.ID, time.Now().UnixNano(), ext)
	dst, err := os.Create(filepath.Join("uploads", name))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not store photo")
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		_ = dst.Close()
		writeError(w, http.StatusInternalServerError, "could not store photo")
		return
	}
	_ = dst.Close()
	photo := models.WorkshopPhoto{
		WorkshopID: ws.ID,
		URL:        "/uploads/" + name,
		Caption:    r.FormValue("caption"),
	}
	if err := h.DB.Create(&photo).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not save photo record")
		return
	}
	writeJSON(w, http.StatusCreated, photo)
}

// listWorkshopPhotos lists a workshop's photos. Public — no auth needed.
func (h *Handler) listWorkshopPhotos(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	ws, ok := h.loadWorkshop(w, r)
	if !ok {
		return
	}
	photos := []models.WorkshopPhoto{}
	if err := h.DB.Where("workshop_id = ?", ws.ID).Find(&photos).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, photos)
}

// deleteWorkshopPhoto deletes one photo: owner only, removes the file
// and the row. Answers 204 with an empty body.
func (h *Handler) deleteWorkshopPhoto(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	ws, ok := h.loadWorkshop(w, r)
	if !ok {
		return
	}
	if ws.UserID != user.ID {
		writeError(w, http.StatusForbidden, "not your workshop")
		return
	}
	photoID, err := strconv.ParseUint(r.PathValue("photoId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid photo id")
		return
	}
	var photo models.WorkshopPhoto
	if err := h.DB.Where("id = ? AND workshop_id = ?", photoID, ws.ID).First(&photo).Error; err != nil {
		writeError(w, http.StatusNotFound, "photo not found")
		return
	}
	_ = os.Remove(filepath.Join("uploads", filepath.Base(photo.URL)))
	if err := h.DB.Delete(&photo).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete photo")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
