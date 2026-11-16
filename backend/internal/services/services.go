package services

import (
	"errors"
	"strings"

	"factorymatch/internal/models"

	"gorm.io/gorm"
)

// WorkshopSearchFilters contains the supported search filters.
type WorkshopSearchFilters struct {
	Process  string `json:"process"`
	Material string `json:"material"`
	District string `json:"district"`
	Keyword  string `json:"keyword"`
}

// WorkshopSearchResult contains a workshop and its capabilities.
type WorkshopSearchResult struct {
	models.Workshop
	Capabilities []models.Capability `json:"capabilities"`
}

// SearchWorkshops searches approved workshops using the supplied filters.
func SearchWorkshops(
	db *gorm.DB,
	filters WorkshopSearchFilters,
) ([]WorkshopSearchResult, error) {

	if db == nil {
		return nil, errors.New("database not configured")
	}

	process := strings.ToLower(strings.TrimSpace(filters.Process))
	material := strings.ToLower(strings.TrimSpace(filters.Material))

	query := db.Model(&models.Workshop{}).
		Where("workshops.verified = ?", true)

	// Filter by manufacturing process (exact, case-insensitive).
	if process != "" || material != "" {
		conditions := []string{
			"capabilities.workshop_id = workshops.id",
		}
		args := []interface{}{}

		if process != "" {
			conditions = append(conditions,
				"LOWER(capabilities.process) = ?")
			args = append(args, process)
		}

		if material != "" {
			conditions = append(conditions,
				"LOWER(capabilities.material) = ?")
			args = append(args, material)
		}

		query = query.Where(
			"EXISTS (SELECT 1 FROM capabilities WHERE "+
				strings.Join(conditions, " AND ")+")",
			args...,
		)
	}

	// Filter by district using existing location fields.
	if district := strings.TrimSpace(filters.District); district != "" {
		pattern := "%" + strings.ToLower(district) + "%"

		query = query.Where(`
			(
				LOWER(workshops.address) LIKE ?
				OR LOWER(workshops.operating_locations) LIKE ?
			)
		`, pattern, pattern)
	}

	// Search workshop profiles and manufacturing capabilities.
	if keyword := strings.TrimSpace(filters.Keyword); keyword != "" {
		pattern := "%" + strings.ToLower(keyword) + "%"

		query = query.Where(`
			(
				LOWER(workshops.name) LIKE ?
				OR LOWER(workshops.description) LIKE ?
				OR LOWER(workshops.address) LIKE ?
				OR LOWER(workshops.operating_locations) LIKE ?
				OR EXISTS (
					SELECT 1
					FROM capabilities
					WHERE capabilities.workshop_id = workshops.id
					AND (
						LOWER(capabilities.process) LIKE ?
						OR LOWER(capabilities.material) LIKE ?
						OR LOWER(capabilities.machine_type) LIKE ?
					)
				)
			)
		`,
			pattern, pattern, pattern, pattern,
			pattern, pattern, pattern,
		)
	}

	// Retrieve matching verified workshops.
	var workshops []models.Workshop

	if err := query.
		Order("workshops.name ASC").
		Find(&workshops).Error; err != nil {
		return nil, err
	}

	results := make([]WorkshopSearchResult, 0, len(workshops))

	if len(workshops) == 0 {
		return results, nil
	}

	// Collect workshop IDs.
	ids := make([]uint, 0, len(workshops))

	for _, workshop := range workshops {
		ids = append(ids, workshop.ID)
	}

	// Retrieve only capabilities matching the applied filters.
	capabilityQuery := db.Where("workshop_id IN ?", ids)

	if process != "" {
		capabilityQuery = capabilityQuery.Where(
			"LOWER(process) = ?", process,
		)
	}

	if material != "" {
		capabilityQuery = capabilityQuery.Where(
			"LOWER(material) = ?", material,
		)
	}

	var capabilities []models.Capability

	if err := capabilityQuery.
		Order("process ASC").
		Find(&capabilities).Error; err != nil {
		return nil, err
	}

	// Group capabilities by workshop.
	capabilityMap := make(map[uint][]models.Capability)

	for _, capability := range capabilities {
		capabilityMap[capability.WorkshopID] = append(
			capabilityMap[capability.WorkshopID],
			capability,
		)
	}

	// Assemble the final search response.
	for _, workshop := range workshops {
		workshopCapabilities := capabilityMap[workshop.ID]

		if workshopCapabilities == nil {
			workshopCapabilities = []models.Capability{}
		}

		results = append(results, WorkshopSearchResult{
			Workshop:     workshop,
			Capabilities: workshopCapabilities,
		})
	}

	return results, nil
}
