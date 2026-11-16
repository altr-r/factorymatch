package handlers

import (
	"net/http"
	"strings"

	"factorymatch/internal/services"
)

// searchWorkshops handles buyer-facing workshop search.
//
// Supported query parameters:
// process, material, district, keyword.
//
// Example:
// GET /api/workshops?process=CNC&material=Steel
func (h *Handler) searchWorkshops(w http.ResponseWriter, r *http.Request) {
	// Ensure the database is configured.
	if !h.requireDB(w) {
		return
	}

	params := r.URL.Query()

	filters := services.WorkshopSearchFilters{
		Process:  strings.TrimSpace(params.Get("process")),
		Material: strings.TrimSpace(params.Get("material")),
		District: strings.TrimSpace(params.Get("district")),
		Keyword:  strings.TrimSpace(params.Get("keyword")),
	}

	// Validate filter lengths.
	filterValues := []string{
		filters.Process,
		filters.Material,
		filters.District,
		filters.Keyword,
	}

	for _, value := range filterValues {
		if len(value) > 100 {
			writeError(
				w,
				http.StatusBadRequest,
				"filter values must be at most 100 characters",
			)
			return
		}
	}

	// Execute the search service.
	results, err := services.SearchWorkshops(h.DB, filters)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			"could not search workshops",
		)
		return
	}

	// Return matching workshops and applied filters.
	writeJSON(w, http.StatusOK, map[string]any{
		"count":     len(results),
		"filters":   filters,
		"workshops": results,
	})
}
