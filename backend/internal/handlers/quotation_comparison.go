// Feature 3.3 — Side-by-Side Quotation Comparison.
// Provides a read-only buyer view of the latest submitted quotation version
// from each workshop for one RFQ. Quotation submission and revisions are
// separate features (3.1 and 3.2).
package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"factorymatch/internal/models"
)

// quotationComparisonItem is one column of the future side-by-side UI.
// Money is represented as an integer number of BDT, as in models.Quotation.
type quotationComparisonItem struct {
	QuotationID      uint      `json:"quotation_id"`
	WorkshopID       uint      `json:"workshop_id"`
	ManufacturerName string    `json:"manufacturer_name"`
	LatestVersion    int       `json:"latest_version"`
	QuotedTotalBDT   int64     `json:"quoted_total_bdt"`
	ProposedQuantity int       `json:"proposed_quantity"`
	LeadTimeDays     int       `json:"lead_time_days"`
	ValidUntil       time.Time `json:"valid_until"`
	Conditions       string    `json:"conditions"`
	SubmittedAt      time.Time `json:"submitted_at"`
	Status           string    `json:"status"` // active, expired, withdrawn
	IsActive         bool      `json:"is_active"`
}

type quotationComparisonResponse struct {
	RFQID             uint                      `json:"rfq_id"`
	RFQTitle          string                    `json:"rfq_title"`
	Currency          string                    `json:"currency"`
	ManufacturerCount int                       `json:"manufacturer_count"`
	ActiveCount       int                       `json:"active_count"`
	Quotations        []quotationComparisonItem `json:"quotations"`
}

// compareQuotations serves GET /api/rfqs/{id}/quotations/compare.
// The route is buyer-only; ownership is independently checked in the query
// to prevent one buyer from viewing a different buyer's confidential quotes.
func (h *Handler) compareQuotations(w http.ResponseWriter, r *http.Request) {
	if !h.requireDB(w) {
		return
	}
	user, ok := currentUser(w, r)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(r.PathValue("id"), 10, 0)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, "invalid RFQ id")
		return
	}

	var rfq models.RFQ
	if err := h.DB.Where("id = ? AND buyer_id = ?", uint(id), user.ID).First(&rfq).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Deliberately identical for an unknown RFQ or one owned by another buyer.
			writeError(w, http.StatusNotFound, "RFQ not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load RFQ")
		return
	}

	// Reading all revisions allows us to pick the most recent version of
	// each workshop without assuming every workshop submitted only one quote.
	var submitted []models.Quotation
	if err := h.DB.Where("rfq_id = ?", rfq.ID).Find(&submitted).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "could not load quotations")
		return
	}
	latest := latestQuotationVersions(submitted)

	workshopIDs := make([]uint, 0, len(latest))
	for _, quote := range latest {
		workshopIDs = append(workshopIDs, quote.WorkshopID)
	}
	workshops := []models.Workshop{}
	if len(workshopIDs) > 0 {
		if err := h.DB.Where("id IN ?", workshopIDs).Find(&workshops).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "could not load manufacturers")
			return
		}
	}

	rows, active, err := buildQuotationComparisonRows(latest, workshops, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not assemble quotation comparison")
		return
	}
	writeJSON(w, http.StatusOK, quotationComparisonResponse{
		RFQID:             rfq.ID,
		RFQTitle:          rfq.Title,
		Currency:          "BDT",
		ManufacturerCount: len(rows),
		ActiveCount:       active,
		Quotations:        rows,
	})
}

// latestQuotationVersions keeps one row per workshop. A withdrawn latest
// version MUST NOT reactivate a previous version. Version is authoritative;
// CreatedAt, then ID, break ties deterministically if data is inconsistent.
func latestQuotationVersions(submitted []models.Quotation) []models.Quotation {
	byWorkshop := make(map[uint]models.Quotation, len(submitted))
	for _, q := range submitted {
		previous, exists := byWorkshop[q.WorkshopID]
		if !exists || q.Version > previous.Version ||
			(q.Version == previous.Version && q.CreatedAt.After(previous.CreatedAt)) ||
			(q.Version == previous.Version && q.CreatedAt.Equal(previous.CreatedAt) && q.ID > previous.ID) {
			byWorkshop[q.WorkshopID] = q
		}
	}

	latest := make([]models.Quotation, 0, len(byWorkshop))
	for _, q := range byWorkshop {
		latest = append(latest, q)
	}
	sort.Slice(latest, func(i, j int) bool { return latest[i].WorkshopID < latest[j].WorkshopID })
	return latest
}

// buildQuotationComparisonRows merges manufacturer identities into the
// latest quotes, including withdrawn and expired ones for buyer transparency.
// Only "active" quotations count as currently usable offers.
func buildQuotationComparisonRows(latest []models.Quotation, workshops []models.Workshop, now time.Time) ([]quotationComparisonItem, int, error) {
	byID := make(map[uint]string, len(workshops))
	for _, ws := range workshops {
		byID[ws.ID] = ws.Name
	}

	rows := make([]quotationComparisonItem, 0, len(latest))
	active := 0
	for _, quote := range latest {
		manufacturer, found := byID[quote.WorkshopID]
		if !found {
			return nil, 0, fmt.Errorf("quotation %d references missing workshop %d", quote.ID, quote.WorkshopID)
		}

		status := "active"
		if quote.Withdrawn {
			status = "withdrawn"
		} else if !quote.ValidUntil.After(now) {
			status = "expired"
		} else {
			active++
		}
		rows = append(rows, quotationComparisonItem{
			QuotationID:      quote.ID,
			WorkshopID:       quote.WorkshopID,
			ManufacturerName: manufacturer,
			LatestVersion:    quote.Version,
			QuotedTotalBDT:   quote.PriceBDT,
			ProposedQuantity: quote.Quantity,
			LeadTimeDays:     quote.LeadTimeDays,
			ValidUntil:       quote.ValidUntil,
			Conditions:       quote.Conditions,
			SubmittedAt:      quote.CreatedAt,
			Status:           status,
			IsActive:         status == "active",
		})
	}

	// Stable order: active offers first, then expired, then withdrawn;
	// cheaper active offers first, with manufacturer name/ID as tie-breakers.
	rank := map[string]int{"active": 0, "expired": 1, "withdrawn": 2}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if rank[a.Status] != rank[b.Status] {
			return rank[a.Status] < rank[b.Status]
		}
		if a.Status == "active" && a.QuotedTotalBDT != b.QuotedTotalBDT {
			return a.QuotedTotalBDT < b.QuotedTotalBDT
		}
		if strings.ToLower(a.ManufacturerName) != strings.ToLower(b.ManufacturerName) {
			return strings.ToLower(a.ManufacturerName) < strings.ToLower(b.ManufacturerName)
		}
		return a.WorkshopID < b.WorkshopID
	})
	return rows, active, nil
}
