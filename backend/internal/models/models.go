// Package models defines the database entities for FactoryMatch.
//
// One struct per feature area, following the functional requirements doc:
// C.1/C.2 (users, verification), Module 1 (profiles, capabilities, search,
// shortlist), Module 2 (RFQs), Module 3 (quotations, orders), Module 4
// (samples, milestones, inspections, deliveries).
package models

import "time"

// Role is the account type: buyer, manufacturer or administrator.
type Role string

const (
	RoleBuyer        Role = "buyer"
	RoleManufacturer Role = "manufacturer"
	RoleAdmin        Role = "admin"
)

// AccountStatus is the current status of an account.
type AccountStatus string

const (
	StatusActive    AccountStatus = "active"
	StatusPending   AccountStatus = "pending"
	StatusSuspended AccountStatus = "suspended"
)

// User is a C.1 account: buyers and manufacturers register themselves;
// admin accounts are created by FactoryMatch, never self-registered.
type User struct {
	ID           uint          `gorm:"primaryKey" json:"id"`
	Name         string        `json:"name"`
	Email        string        `gorm:"uniqueIndex" json:"email"`
	PasswordHash string        `json:"-"`
	Role         Role          `json:"role"`
	Status       AccountStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// OrganizationProfile is the 1.1 buyer organization profile.
type OrganizationProfile struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"uniqueIndex" json:"user_id"`
	BusinessName   string    `json:"business_name"`
	ContactPerson  string    `json:"contact_person"`
	ContactDetails string    `json:"contact_details"`
	District       string    `json:"district"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Workshop is the 1.2 manufacturer workshop profile. Verified is set by an
// admin (C.2); only verified workshops appear in search (1.4) and are
// eligible for RFQ invitations (2.3).
type Workshop struct {
	ID                 uint   `gorm:"primaryKey" json:"id"`
	UserID             uint   `gorm:"index" json:"user_id"`
	Name               string `json:"name"`
	Address            string `json:"address"`
	ContactDetails     string `json:"contact_details"`
	OperatingLocations string `json:"operating_locations"`
	Description        string `json:"description"`
	Verified           bool   `json:"verified"`
	// VerificationStatus is the C.2 review state: pending, under-review,
	// correction-requested or approved. VerificationNote records the admin's
	// reason (e.g. why a correction was requested).
	VerificationStatus string    `json:"verification_status"`
	VerificationNote   string    `json:"verification_note"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// WorkshopPhoto is a 1.2 photograph of previous work.
type WorkshopPhoto struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	WorkshopID uint      `gorm:"index" json:"workshop_id"`
	URL        string    `json:"url"`
	Caption    string    `json:"caption"`
	CreatedAt  time.Time `json:"created_at"`
}

// Capability is one 1.3 catalogue entry: a supported process/material/
// machine combination with its dimensions, MOQ and lead time, stored
// structured (not free text) so it can drive search and matching.
type Capability struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	WorkshopID   uint      `gorm:"index" json:"workshop_id"`
	Process      string    `json:"process"`       // e.g. CNC machining, metal fabrication
	Material     string    `json:"material"`      // e.g. stainless steel, aluminium
	MachineType  string    `json:"machine_type"`  // e.g. 3-axis CNC mill
	MinDimension string    `json:"min_dimension"` // feasible dimensions, free-form but structured fields
	MaxDimension string    `json:"max_dimension"`
	MOQ          int       `json:"moq"` // minimum order quantity
	LeadTimeDays int       `json:"lead_time_days"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Shortlist is a 1.5 saved workshop on a buyer's personal shortlist.
type Shortlist struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	BuyerID    uint      `gorm:"uniqueIndex:idx_shortlist_buyer_workshop" json:"buyer_id"`
	WorkshopID uint      `gorm:"uniqueIndex:idx_shortlist_buyer_workshop" json:"workshop_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// RFQStatus tracks a 2.5 RFQ through its lifecycle.
type RFQStatus string

const (
	RFQDraft     RFQStatus = "draft"
	RFQOpen      RFQStatus = "open"
	RFQClosed    RFQStatus = "closed"
	RFQAwarded   RFQStatus = "awarded"
	RFQCancelled RFQStatus = "cancelled"
)

// RFQ is a 2.1 manufacturing request for quotation.
type RFQ struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	BuyerID             uint      `gorm:"index" json:"buyer_id"`
	Title               string    `json:"title"`
	Process             string    `json:"process"`
	Material            string    `json:"material"`
	Quantity            int       `json:"quantity"`
	Dimensions          string    `json:"dimensions"`
	Tolerances          string    `json:"tolerances"`
	SpecialInstructions string    `json:"special_instructions"`
	DeliveryLocation    string    `json:"delivery_location"`
	Deadline            time.Time `json:"deadline"`
	Status              RFQStatus `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// RFQDrawing is a 2.2 technical drawing / spec file attached to an RFQ,
// kept with numbered revisions.
type RFQDrawing struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RFQID     uint      `gorm:"index" json:"rfq_id"`
	FileURL   string    `json:"file_url"`
	Revision  int       `json:"revision"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// InvitationStatus is the 2.3 state of an RFQ invitation.
type InvitationStatus string

const (
	InvitePending  InvitationStatus = "pending"
	InviteAccepted InvitationStatus = "accepted"
	InviteDeclined InvitationStatus = "declined"
)

// RFQInvitation records a 2.3 invitation of a workshop to an RFQ.
type RFQInvitation struct {
	ID         uint             `gorm:"primaryKey" json:"id"`
	RFQID      uint             `gorm:"index" json:"rfq_id"`
	WorkshopID uint             `gorm:"index" json:"workshop_id"`
	Status     InvitationStatus `json:"status"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// Clarification is one 2.4 message in an RFQ clarification thread.
type Clarification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RFQID     uint      `gorm:"index" json:"rfq_id"`
	SenderID  uint      `json:"sender_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// Quotation is a 3.1 manufacturer quotation against an open RFQ.
// Revisions (3.2) are new rows with a higher Version.
type Quotation struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RFQID        uint      `gorm:"index" json:"rfq_id"`
	WorkshopID   uint      `gorm:"index" json:"workshop_id"`
	Quantity     int       `json:"quantity"`
	PriceBDT     int64     `json:"price_bdt"` // total price in BDT (poisha? agree as a team; use BDT here)
	LeadTimeDays int       `json:"lead_time_days"`
	ValidUntil   time.Time `json:"valid_until"`
	Conditions   string    `json:"conditions"`
	Version      int       `json:"version"`
	Withdrawn    bool      `json:"withdrawn"`
	CreatedAt    time.Time `json:"created_at"`
}

// ManufacturingOrder is a 3.5 order generated from an awarded quotation.
type ManufacturingOrder struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	RFQID                uint      `gorm:"index" json:"rfq_id"`
	QuotationID          uint      `json:"quotation_id"`
	BuyerID              uint      `gorm:"index" json:"buyer_id"`
	WorkshopID           uint      `gorm:"index" json:"workshop_id"`
	PriceBDT             int64     `json:"price_bdt"`
	Quantity             int       `json:"quantity"`
	SpecVersion          int       `json:"spec_version"`
	DeliveryRequirements string    `json:"delivery_requirements"`
	AgreedDates          string    `json:"agreed_dates"`
	BuyerConfirmed       bool      `json:"buyer_confirmed"`
	WorkshopConfirmed    bool      `json:"workshop_confirmed"`
	Status               string    `json:"status"` // e.g. awaiting-confirmation, in-production, delivered
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Sample is a 4.1 prototype/sample submission for buyer approval.
type Sample struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	OrderID       uint      `gorm:"index" json:"order_id"`
	PhotoURLs     string    `json:"photo_urls"` // comma-separated for now; split into a table if needed
	Measurements  string    `json:"measurements"`
	Notes         string    `json:"notes"`
	Status        string    `json:"status"` // submitted, approved, rejected
	ReviewComment string    `json:"review_comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Milestone is a 4.2 planned production stage.
type Milestone struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	OrderID    uint       `gorm:"index" json:"order_id"`
	Stage      string     `json:"stage"` // e.g. material preparation, fabrication, finishing
	TargetDate time.Time  `json:"target_date"`
	ActualDate *time.Time `json:"actual_date"`
	Status     string     `json:"status"` // planned, in-progress, done, overdue
	Evidence   string     `json:"evidence"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Inspection is a 4.3 quality inspection record.
type Inspection struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	OrderID    uint      `gorm:"index" json:"order_id"`
	Checklist  string    `json:"checklist"`
	Results    string    `json:"results"`
	PhotoURLs  string    `json:"photo_urls"`
	IssueNotes string    `json:"issue_notes"`
	Status     string    `json:"status"` // recorded, flagged, resolved
	Resolution string    `json:"resolution"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Delivery is a 4.4 dispatch / acceptance record.
type Delivery struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrderID        uint      `gorm:"index" json:"order_id"`
	DispatchDate   time.Time `json:"dispatch_date"`
	DeliveredQty   int       `json:"delivered_quantity"`
	Reference      string    `json:"delivery_reference"`
	Acceptance     string    `json:"acceptance"` // full, partial, issue
	OutstandingQty int       `json:"outstanding_quantity"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
