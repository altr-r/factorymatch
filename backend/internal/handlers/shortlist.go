// 1.5 Workshop Shortlisting.
//
// Buyer shortlist: save verified workshops, review them, and pick them as
// RFQ recipients. Model (Shortlist) already lives in
// internal/models/models.go — nothing to add there. RFQ creation itself
// is Module 2, not here.
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"gorm.io/gorm"

	"factorymatch/internal/models"
)

// listShortlist returns the buyer's shortlisted workshops with their
// details. Empty shortlist → empty array, not null.
func (h *Handler) listShortlist(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	var entries []models.Shortlist
	if err := h.DB.Where("buyer_id = ?", user.ID).Find(&entries).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	ids := make([]uint, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.WorkshopID)
	}
	workshops := []models.Workshop{}
	if len(ids) > 0 {
		_ = h.DB.Where("id IN ?", ids).Find(&workshops).Error
	}
	writeJSON(w, http.StatusOK, workshops)
}

type shortlistRequest struct {
	WorkshopID uint `json:"workshop_id"`
}

// addShortlist saves a workshop to the buyer's personal shortlist.
// Only verified workshops can be shortlisted. Saving the same workshop
// twice does not create a duplicate — it returns the existing entry
// with 200 instead of 201.
func (h *Handler) addShortlist(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	var req shortlistRequest
	if !readJSON(w, r, &req) {
		return
	}
	var ws models.Workshop
	if err := h.DB.First(&ws, req.WorkshopID).Error; err != nil {
		writeError(w, http.StatusNotFound, "workshop not found")
		return
	}
	if !ws.Verified {
		writeError(w, http.StatusBadRequest, "only verified workshops can be shortlisted")
		return
	}
	var existing models.Shortlist
	if err := h.DB.Where("buyer_id = ? AND workshop_id = ?", user.ID, req.WorkshopID).First(&existing).Error; err == nil {
		writeJSON(w, http.StatusOK, existing)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	entry := models.Shortlist{BuyerID: user.ID, WorkshopID: req.WorkshopID}
	if err := h.DB.Create(&entry).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not save shortlist entry")
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

// removeShortlist removes a workshop from the buyer's shortlist.
// Answers 204 with an empty body; 404 when it was not shortlisted.
func (h *Handler) removeShortlist(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	workshopID, err := strconv.ParseUint(r.PathValue("workshopId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workshop id")
		return
	}
	res := h.DB.Where("buyer_id = ? AND workshop_id = ?", user.ID, workshopID).Delete(&models.Shortlist{})
	if res.Error != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if res.RowsAffected == 0 {
		writeError(w, http.StatusNotFound, "workshop not in shortlist")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// shortlistRecipients returns shortlisted workshops as pickable RFQ
// recipients (1.5.3): just id + name each. RFQ creation itself is
// Module 2.
func (h *Handler) shortlistRecipients(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}
	var entries []models.Shortlist
	if err := h.DB.Where("buyer_id = ?", user.ID).Find(&entries).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	ids := make([]uint, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.WorkshopID)
	}
	type recipient struct {
		WorkshopID uint   `json:"workshop_id"`
		Name       string `json:"name"`
	}
	recipients := []recipient{}
	if len(ids) > 0 {
		var workshops []models.Workshop
		_ = h.DB.Where("id IN ?", ids).Find(&workshops).Error
		for _, ws := range workshops {
			recipients = append(recipients, recipient{WorkshopID: ws.ID, Name: ws.Name})
		}
	}
	writeJSON(w, http.StatusOK, recipients)
}
