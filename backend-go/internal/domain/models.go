package domain

import "time"

type Role string

const (
	RoleConsumer Role = "CONSUMER"
	RoleMerchant Role = "MERCHANT"
	RoleRider    Role = "RIDER"
	RoleAdmin    Role = "ADMIN"
)

func (r Role) Valid() bool {
	switch r {
	case RoleConsumer, RoleMerchant, RoleRider, RoleAdmin:
		return true
	default:
		return false
	}
}

type MerchantStatus string

const (
	MerchantPending   MerchantStatus = "PENDING"
	MerchantApproved  MerchantStatus = "APPROVED"
	MerchantRejected  MerchantStatus = "REJECTED"
	MerchantSuspended MerchantStatus = "SUSPENDED"
)

type DemandStatus string

const (
	DemandPendingReview DemandStatus = "PENDING_REVIEW"
	DemandOpen          DemandStatus = "OPEN"
	DemandReady         DemandStatus = "READY"
	DemandClosed        DemandStatus = "CLOSED"
	DemandRejected      DemandStatus = "REJECTED"
)

type OfferStatus string

const (
	OfferSubmitted OfferStatus = "SUBMITTED"
	OfferAccepted  OfferStatus = "ACCEPTED"
	OfferRejected  OfferStatus = "REJECTED"
	OfferWithdrawn OfferStatus = "WITHDRAWN"
)

type CampaignStatus string

const (
	CampaignDraft         CampaignStatus = "DRAFT"
	CampaignPendingReview CampaignStatus = "PENDING_REVIEW"
	CampaignOpen          CampaignStatus = "OPEN"
	CampaignSoldOut       CampaignStatus = "SOLD_OUT"
	CampaignClosed        CampaignStatus = "CLOSED"
	CampaignCancelled     CampaignStatus = "CANCELLED"
)

type OrderStatus string

const (
	OrderPendingPayment OrderStatus = "PENDING_PAYMENT"
	OrderPaid           OrderStatus = "PAID"
	OrderAccepted       OrderStatus = "ACCEPTED"
	OrderPreparing      OrderStatus = "PREPARING"
	OrderReadyForPickup OrderStatus = "READY_FOR_PICKUP"
	OrderPickedUp       OrderStatus = "PICKED_UP"
	OrderDelivering     OrderStatus = "DELIVERING"
	OrderDelivered      OrderStatus = "DELIVERED"
	OrderCancelled      OrderStatus = "CANCELLED"
	OrderRefunded       OrderStatus = "REFUNDED"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionUser struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Role       Role   `json:"role"`
	MerchantID string `json:"merchant_id,omitempty"`
}

type Merchant struct {
	ID           string         `json:"id"`
	OwnerUserID  string         `json:"owner_user_id"`
	Name         string         `json:"name"`
	Status       MerchantStatus `json:"status"`
	License      map[string]any `json:"license"`
	ReviewReason string         `json:"review_reason,omitempty"`
	ReviewedAt   *time.Time     `json:"reviewed_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type DemandSpec struct {
	Category        string   `json:"category"`
	Title           string   `json:"title"`
	ServiceArea     string   `json:"service_area"`
	ServingDate     string   `json:"serving_date"`
	ServingTime     string   `json:"serving_time"`
	BudgetMinCents  int64    `json:"budget_min_cents"`
	BudgetMaxCents  int64    `json:"budget_max_cents"`
	Quantity        int      `json:"quantity"`
	WeightMinGrams  int      `json:"weight_min_grams"`
	WeightMaxGrams  int      `json:"weight_max_grams"`
	HardConstraints []string `json:"hard_constraints"`
	Preferences     []string `json:"preferences"`
	Notes           string   `json:"notes,omitempty"`
}

type Demand struct {
	DemandSpec
	ID             string       `json:"id"`
	CreatedBy      string       `json:"created_by"`
	MinimumMembers int          `json:"minimum_members"`
	MaximumMembers int          `json:"maximum_members"`
	MemberCount    int          `json:"member_count"`
	Status         DemandStatus `json:"status"`
	ReviewedBy     string       `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time   `json:"reviewed_at,omitempty"`
	ReviewReason   string       `json:"review_reason,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

type DemandMember struct {
	ID          string    `json:"id"`
	DemandID    string    `json:"demand_id"`
	UserID      string    `json:"user_id,omitempty"`
	Quantity    int       `json:"quantity"`
	WeightGrams int       `json:"weight_grams,omitempty"`
	Preferences []string  `json:"preferences"`
	Notes       string    `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Offer struct {
	ID                  string      `json:"id"`
	DemandID            string      `json:"demand_id"`
	MerchantID          string      `json:"merchant_id"`
	UnitPriceCents      int64       `json:"unit_price_cents"`
	ProductionCapacity  int         `json:"production_capacity"`
	WeightGrams         int         `json:"weight_grams"`
	Ingredients         []string    `json:"ingredients"`
	Allergens           []string    `json:"allergens"`
	OilLevel            string      `json:"oil_level"`
	SaltLevel           string      `json:"salt_level"`
	ProductionTime      string      `json:"production_time"`
	ShelfLifeMinutes    int         `json:"shelf_life_minutes"`
	StorageInstructions string      `json:"storage_instructions"`
	Notes               string      `json:"notes,omitempty"`
	Status              OfferStatus `json:"status"`
	CreatedAt           time.Time   `json:"created_at"`
}

type FoodSpec struct {
	WeightGrams         int      `json:"weight_grams"`
	Ingredients         []string `json:"ingredients"`
	Allergens           []string `json:"allergens"`
	OilLevel            string   `json:"oil_level"`
	SaltLevel           string   `json:"salt_level"`
	ProductionTime      string   `json:"production_time"`
	ShelfLifeMinutes    int      `json:"shelf_life_minutes"`
	StorageInstructions string   `json:"storage_instructions"`
}

type Campaign struct {
	ID               string         `json:"id"`
	DemandID         string         `json:"demand_id"`
	OfferID          string         `json:"offer_id"`
	MerchantID       string         `json:"merchant_id"`
	Title            string         `json:"title"`
	Description      string         `json:"description,omitempty"`
	UnitPriceCents   int64          `json:"unit_price_cents"`
	DeliveryFeeCents int64          `json:"delivery_fee_cents"`
	PlatformFeeBPS   int64          `json:"platform_fee_bps"`
	MinimumOrders    int            `json:"minimum_orders"`
	MaximumOrders    int            `json:"maximum_orders"`
	CurrentOrders    int            `json:"current_orders"`
	StartsAt         time.Time      `json:"starts_at"`
	EndsAt           time.Time      `json:"ends_at"`
	PickupPoint      string         `json:"pickup_point"`
	FoodSpec         FoodSpec       `json:"food_spec"`
	Status           CampaignStatus `json:"status"`
	ReviewReason     string         `json:"review_reason,omitempty"`
	ReviewedAt       *time.Time     `json:"reviewed_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type Order struct {
	ID               string      `json:"id"`
	CampaignID       string      `json:"campaign_id"`
	ConsumerID       string      `json:"consumer_id"`
	Quantity         int         `json:"quantity"`
	DeliveryAddress  string      `json:"delivery_address"`
	ContactName      string      `json:"contact_name"`
	ContactPhone     string      `json:"contact_phone"`
	Status           OrderStatus `json:"status"`
	UnitPriceCents   int64       `json:"unit_price_cents"`
	SubtotalCents    int64       `json:"subtotal_cents"`
	DeliveryFeeCents int64       `json:"delivery_fee_cents"`
	PlatformFeeCents int64       `json:"platform_fee_cents"`
	TotalCents       int64       `json:"total_cents"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type IdempotencyRecord struct {
	ActorID     string
	Key         string
	Fingerprint string
	Status      int
	Response    []byte
	CreatedAt   time.Time
}
