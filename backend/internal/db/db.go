package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"factorymatch/internal/models"
)


func Open(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, nil
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}


func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.OrganizationProfile{},
		&models.Workshop{},
		&models.WorkshopPhoto{},
		&models.Capability{},
		&models.Shortlist{},
		&models.RFQ{},
		&models.RFQDrawing{},
		&models.RFQInvitation{},
		&models.Clarification{},
		&models.Quotation{},
		&models.ManufacturingOrder{},
		&models.Sample{},
		&models.Milestone{},
		&models.Inspection{},
		&models.Delivery{},
	)
}