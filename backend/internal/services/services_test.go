package services

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"factorymatch/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openTestDB connects to the dedicated PostgreSQL test database.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("could not connect to test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("could not access SQL database: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func hasWorkshop(results []WorkshopSearchResult, id uint) bool {
	for _, result := range results {
		if result.ID == id {
			return true
		}
	}
	return false
}

func getWorkshop(
	results []WorkshopSearchResult,
	id uint,
) (WorkshopSearchResult, bool) {
	for _, result := range results {
		if result.ID == id {
			return result, true
		}
	}
	return WorkshopSearchResult{}, false
}

// TestSearchWorkshops verifies the FAC-3 search requirements.
func TestSearchWorkshops(t *testing.T) {
	database := openTestDB(t)

	// All test records are rolled back after testing.
	tx := database.Begin()
	if tx.Error != nil {
		t.Fatalf("could not begin transaction: %v", tx.Error)
	}
	defer tx.Rollback()

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)

	process := "FAC3 CNC " + suffix
	otherProcess := "FAC3 Laser " + suffix
	material := "FAC3 Steel " + suffix
	otherMaterial := "FAC3 Stainless Steel " + suffix
	district := "FAC3 District " + suffix
	keyword := "FAC3 Keyword " + suffix

	// Create four workshops.
	workshops := []models.Workshop{
		{
			Name:               "FAC3 Workshop A " + suffix,
			Address:            district + " Road",
			OperatingLocations: "Other Location",
			Description:        "Description " + keyword,
			Verified:           true,
			VerificationStatus: "approved",
		},
		{
			Name:               "FAC3 Workshop B " + suffix,
			Address:            "Different Address",
			OperatingLocations: district,
			Description:        "Another description",
			Verified:           true,
			VerificationStatus: "approved",
		},
		{
			Name:               "FAC3 Pending Workshop " + suffix,
			Address:            district,
			OperatingLocations: district,
			Description:        keyword,
			Verified:           false,
			VerificationStatus: "pending",
		},
		{
			Name:               "FAC3 Workshop D " + suffix,
			Address:            "Unrelated Address",
			OperatingLocations: "Unrelated Location",
			Description:        "Unrelated description",
			Verified:           true,
			VerificationStatus: "approved",
		},
	}

	for i := range workshops {
		if err := tx.Create(&workshops[i]).Error; err != nil {
			t.Fatalf("could not create workshop: %v", err)
		}
	}

	// A has one matching capability and one non-matching material.
	// B has process and material on DIFFERENT capabilities.
	// C is pending and must never appear.
	// D is a material partial-match decoy.
	capabilities := []models.Capability{
		{
			WorkshopID:  workshops[0].ID,
			Process:     process,
			Material:    material,
			MachineType: "Matching Machine",
		},
		{
			WorkshopID:  workshops[0].ID,
			Process:     process,
			Material:    otherMaterial,
			MachineType: "Second Machine",
		},
		{
			WorkshopID:  workshops[1].ID,
			Process:     process,
			Material:    otherMaterial,
			MachineType: "Machine " + keyword,
		},
		{
			WorkshopID:  workshops[1].ID,
			Process:     otherProcess,
			Material:    material,
			MachineType: "Different Machine",
		},
		{
			WorkshopID:  workshops[2].ID,
			Process:     process,
			Material:    material,
			MachineType: "Pending Machine",
		},
		{
			WorkshopID:  workshops[3].ID,
			Process:     otherProcess,
			Material:    otherMaterial,
			MachineType: "Decoy Machine",
		},
	}

	for i := range capabilities {
		if err := tx.Create(&capabilities[i]).Error; err != nil {
			t.Fatalf("could not create capability: %v", err)
		}
	}

	// 1. Only verified workshops appear.
	t.Run("approved_only", func(t *testing.T) {
		results, err := SearchWorkshops(
			tx, WorkshopSearchFilters{},
		)
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		for _, id := range []uint{
			workshops[0].ID,
			workshops[1].ID,
			workshops[3].ID,
		} {
			if !hasWorkshop(results, id) {
				t.Errorf("approved workshop %d was not returned", id)
			}
		}

		if hasWorkshop(results, workshops[2].ID) {
			t.Error("pending workshop appeared in search results")
		}
	})

	// 2. Process filtering is case-insensitive.
	t.Run("process_filter", func(t *testing.T) {
		results, err := SearchWorkshops(tx, WorkshopSearchFilters{
			Process: strings.ToUpper(process),
		})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		if !hasWorkshop(results, workshops[0].ID) ||
			!hasWorkshop(results, workshops[1].ID) {
			t.Error("expected workshops with matching process")
		}

		if hasWorkshop(results, workshops[2].ID) ||
			hasWorkshop(results, workshops[3].ID) {
			t.Error("unexpected workshop returned for process filter")
		}
	})

	// 3. Material filtering uses exact matching.
	t.Run("material_filter", func(t *testing.T) {
		results, err := SearchWorkshops(tx, WorkshopSearchFilters{
			Material: strings.ToUpper(material),
		})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		if !hasWorkshop(results, workshops[0].ID) ||
			!hasWorkshop(results, workshops[1].ID) {
			t.Error("expected workshops with matching material")
		}

		if hasWorkshop(results, workshops[2].ID) ||
			hasWorkshop(results, workshops[3].ID) {
			t.Error("unexpected workshop returned for material filter")
		}
	})

	// 4. District matches address or operating locations.
	t.Run("district_filter", func(t *testing.T) {
		results, err := SearchWorkshops(tx, WorkshopSearchFilters{
			District: district,
		})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		if !hasWorkshop(results, workshops[0].ID) ||
			!hasWorkshop(results, workshops[1].ID) {
			t.Error("expected district matches were not returned")
		}

		if hasWorkshop(results, workshops[2].ID) ||
			hasWorkshop(results, workshops[3].ID) {
			t.Error("unexpected workshop returned for district filter")
		}
	})

	// 5. Keyword searches profiles and capabilities.
	t.Run("keyword_filter", func(t *testing.T) {
		results, err := SearchWorkshops(tx, WorkshopSearchFilters{
			Keyword: strings.ToLower(keyword),
		})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		if !hasWorkshop(results, workshops[0].ID) ||
			!hasWorkshop(results, workshops[1].ID) {
			t.Error("expected keyword matches were not returned")
		}

		if hasWorkshop(results, workshops[2].ID) ||
			hasWorkshop(results, workshops[3].ID) {
			t.Error("unexpected workshop returned for keyword filter")
		}
	})

	// 6. Process and material must match the SAME capability.
	t.Run("combined_filters_same_capability", func(t *testing.T) {
		results, err := SearchWorkshops(tx, WorkshopSearchFilters{
			Process:  process,
			Material: material,
		})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		if !hasWorkshop(results, workshops[0].ID) {
			t.Error("workshop A should match both filters")
		}

		if hasWorkshop(results, workshops[1].ID) {
			t.Error("workshop B matched process and material on different capabilities")
		}

		if hasWorkshop(results, workshops[2].ID) ||
			hasWorkshop(results, workshops[3].ID) {
			t.Error("unexpected workshop returned")
		}

		// Only the capability matching both filters should be returned.
		result, found := getWorkshop(results, workshops[0].ID)
		if !found {
			t.Fatal("matching workshop was not returned")
		}

		if len(result.Capabilities) != 1 {
			t.Fatalf(
				"expected 1 matching capability, got %d",
				len(result.Capabilities),
			)
		}

		if result.Capabilities[0].Material != material {
			t.Error("returned capability has an incorrect material")
		}
	})

	// 7. A search with no matches returns an empty result.
	t.Run("no_matching_workshops", func(t *testing.T) {
		results, err := SearchWorkshops(tx, WorkshopSearchFilters{
			Process: "FAC3 nonexistent " + suffix,
		})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}
