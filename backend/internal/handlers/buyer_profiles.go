package handlers

import (
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"factorymatch/internal/models"
)

// buyerProfileRequest contains the fields accepted
// when creating or updating a buyer organization profile.
type buyerProfileRequest struct {
	BusinessName   string `json:"business_name"`
	ContactPerson  string `json:"contact_person"`
	ContactDetails string `json:"contact_details"`
	District       string `json:"district"`
	Description    string `json:"description"`
}

// readBuyerProfileRequest decodes and validates the request body.
func readBuyerProfileRequest(w http.ResponseWriter, r *http.Request) (buyerProfileRequest, bool) {
	var req buyerProfileRequest

	if !readJSON(w, r, &req) {
		return req, false
	}

	req.BusinessName = strings.TrimSpace(req.BusinessName)
	req.ContactPerson = strings.TrimSpace(req.ContactPerson)
	req.ContactDetails = strings.TrimSpace(req.ContactDetails)
	req.District = strings.TrimSpace(req.District)
	req.Description = strings.TrimSpace(req.Description)

	if req.BusinessName == "" ||
		req.ContactPerson == "" ||
		req.ContactDetails == "" ||
		req.District == "" ||
		req.Description == "" {
		writeError(w, http.StatusBadRequest, "all buyer profile fields are required")
		return req, false
	}

	return req, true
}

// createBuyerProfile creates an organization profile
// for the authenticated buyer.
func (h *Handler) createBuyerProfile(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}

	user, ok := currentUser(w, r)
	if !ok {
		return
	}

	req, ok := readBuyerProfileRequest(w, r)
	if !ok {
		return
	}

	var existing models.OrganizationProfile

	err := h.DB.Where("user_id = ?", user.ID).First(&existing).Error

	if err == nil {
		writeError(w, http.StatusConflict, "buyer profile already exists")
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	profile := models.OrganizationProfile{
		UserID:         user.ID,
		BusinessName:   req.BusinessName,
		ContactPerson:  req.ContactPerson,
		ContactDetails: req.ContactDetails,
		District:       req.District,
		Description:    req.Description,
	}

	if err := h.DB.Create(&profile).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not create buyer profile")
		return
	}

	writeJSON(w, http.StatusCreated, profile)
}

// getBuyerProfile returns the authenticated buyer's organization profile.
func (h *Handler) getBuyerProfile(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}

	user, ok := currentUser(w, r)
	if !ok {
		return
	}

	var profile models.OrganizationProfile

	if err := h.DB.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "buyer profile not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// updateBuyerProfile updates the authenticated buyer's own profile.
func (h *Handler) updateBuyerProfile(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}

	user, ok := currentUser(w, r)
	if !ok {
		return
	}

	req, ok := readBuyerProfileRequest(w, r)
	if !ok {
		return
	}

	var profile models.OrganizationProfile

	if err := h.DB.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "buyer profile not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	profile.BusinessName = req.BusinessName
	profile.ContactPerson = req.ContactPerson
	profile.ContactDetails = req.ContactDetails
	profile.District = req.District
	profile.Description = req.Description

	if err := h.DB.Save(&profile).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not update buyer profile")
		return
	}

	writeJSON(w, http.StatusOK, profile)
}
