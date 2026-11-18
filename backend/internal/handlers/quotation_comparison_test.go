package handlers

import (
	"testing"
	"time"

	"factorymatch/internal/models"
)

func TestLatestQuotationVersions(t *testing.T) {
	now := time.Date(2026, time.November, 18, 1, 0, 0, 0, time.UTC)
	quotes := []models.Quotation{
		{ID: 10, WorkshopID: 2, Version: 1, PriceBDT: 50000, CreatedAt: now.Add(-3 * time.Hour)},
		{ID: 20, WorkshopID: 1, Version: 1, PriceBDT: 40000, CreatedAt: now.Add(-4 * time.Hour)},
		{ID: 11, WorkshopID: 2, Version: 2, Withdrawn: true, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: 21, WorkshopID: 1, Version: 2, PriceBDT: 38000, CreatedAt: now.Add(-time.Hour)},
		// A version-1 row with a newer timestamp must not replace version 2.
		{ID: 22, WorkshopID: 1, Version: 1, PriceBDT: 37000, CreatedAt: now},
	}
	latest := latestQuotationVersions(quotes)
	if len(latest) != 2 {
		t.Fatalf("want 2 latest quotations; got %d", len(latest))
	}
	if latest[0].WorkshopID != 1 || latest[0].ID != 21 {
		t.Errorf("workshop 1 latest: got %+v; want quotation 21", latest[0])
	}
	if latest[1].WorkshopID != 2 || latest[1].ID != 11 || !latest[1].Withdrawn {
		t.Errorf("workshop 2 latest withdrawal: got %+v", latest[1])
	}
}

func TestLatestQuotationVersionsTieBreakers(t *testing.T) {
	now := time.Date(2026, 11, 18, 1, 0, 0, 0, time.UTC)
	quotes := []models.Quotation{
		{ID: 3, WorkshopID: 7, Version: 2, CreatedAt: now},
		{ID: 9, WorkshopID: 7, Version: 2, CreatedAt: now},
		{ID: 6, WorkshopID: 7, Version: 2, CreatedAt: now.Add(time.Minute)},
		{ID: 1, WorkshopID: 7, Version: 1, CreatedAt: now.Add(2 * time.Minute)},
	}
	got := latestQuotationVersions(quotes)
	if len(got) != 1 || got[0].ID != 6 {
		t.Fatalf("unexpected tie resolution: %+v", got)
	}
	if result := latestQuotationVersions(nil); result == nil || len(result) != 0 {
		t.Fatalf("empty input should return an empty slice: %#v", result)
	}
}

func TestBuildQuotationComparisonRows(t *testing.T) {
	now := time.Date(2026, 11, 18, 1, 0, 0, 0, time.UTC)
	quotes := []models.Quotation{
		{ID: 1, WorkshopID: 1, Version: 3, PriceBDT: 51000, Quantity: 100, LeadTimeDays: 12, Conditions: "50% advance", ValidUntil: now.Add(time.Hour)},
		{ID: 2, WorkshopID: 2, Version: 1, PriceBDT: 20000, ValidUntil: now}, // expires at the exact instant
		{ID: 3, WorkshopID: 3, Version: 2, PriceBDT: 10000, Withdrawn: true, ValidUntil: now.Add(time.Hour)},
		{ID: 4, WorkshopID: 4, Version: 1, PriceBDT: 45000, ValidUntil: now.Add(48 * time.Hour)},
	}
	workshops := []models.Workshop{
		{ID: 1, Name: "Zeta Fab"}, {ID: 2, Name: "Old Quotes"},
		{ID: 3, Name: "Withdrawn Inc"}, {ID: 4, Name: "Alpha CNC"},
	}
	rows, active, err := buildQuotationComparisonRows(quotes, workshops, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 || active != 2 {
		t.Fatalf("want 4 total, 2 active: rows=%d active=%d", len(rows), active)
	}
	if rows[0].WorkshopID != 4 || rows[0].Status != "active" || rows[0].QuotedTotalBDT != 45000 {
		t.Errorf("cheapest active should be first: %+v", rows[0])
	}
	if rows[1].WorkshopID != 1 || rows[1].LatestVersion != 3 || rows[1].ManufacturerName != "Zeta Fab" || rows[1].Conditions != "50% advance" || rows[1].ProposedQuantity != 100 {
		t.Errorf("active quote not mapped correctly: %+v", rows[1])
	}
	if rows[2].WorkshopID != 2 || rows[2].Status != "expired" || rows[2].IsActive {
		t.Errorf("expired quote incorrectly active: %+v", rows[2])
	}
	if rows[3].WorkshopID != 3 || rows[3].Status != "withdrawn" || rows[3].IsActive {
		t.Errorf("withdrawn quote incorrectly active: %+v", rows[3])
	}
}

func TestBuildQuotationComparisonRowsEmptyAndMissingWorkshop(t *testing.T) {
	now := time.Now()
	rows, active, err := buildQuotationComparisonRows(nil, nil, now)
	if err != nil || rows == nil || len(rows) != 0 || active != 0 {
		t.Fatalf("empty comparison: rows=%#v active=%d err=%v", rows, active, err)
	}
	_, _, err = buildQuotationComparisonRows([]models.Quotation{{ID: 1, WorkshopID: 5}}, nil, now)
	if err == nil {
		t.Fatal("missing manufacturer must not silently produce an incomplete comparison")
	}
}
