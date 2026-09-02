package store

import (
	"context"
	"errors"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/domain"
)

var (
	ErrNotFound  = errors.New("resource not found")
	ErrConflict  = errors.New("resource conflict")
	ErrForbidden = errors.New("forbidden")
	ErrInvalid   = errors.New("invalid resource state")
)

type CreateUserInput struct {
	Name        string
	Role        domain.Role
	ExternalKey string
}

type CreateMerchantInput struct {
	OwnerUserID string
	Name        string
	License     map[string]any
}

type CreateDemandInput struct {
	CreatedBy      string
	MinimumMembers int
	MaximumMembers int
	Spec           domain.DemandSpec
}

type CreateMemberInput struct {
	DemandID    string
	UserID      string
	Quantity    int
	WeightGrams int
	Preferences []string
	Notes       string
}

type CreateOfferInput struct {
	DemandID            string
	MerchantID          string
	UnitPriceCents      int64
	ProductionCapacity  int
	WeightGrams         int
	Ingredients         []string
	Allergens           []string
	OilLevel            string
	SaltLevel           string
	ProductionTime      string
	ShelfLifeMinutes    int
	StorageInstructions string
	Notes               string
}

type CreateCampaignInput struct {
	DemandID         string
	OfferID          string
	MerchantID       string
	Title            string
	Description      string
	UnitPriceCents   int64
	DeliveryFeeCents int64
	PlatformFeeBPS   int64
	MinimumOrders    int
	MaximumOrders    int
	StartsAt         time.Time
	EndsAt           time.Time
	PickupPoint      string
	FoodSpec         domain.FoodSpec
}

type CreateOrderInput struct {
	CampaignID      string
	ConsumerID      string
	Quantity        int
	DeliveryAddress string
	ContactName     string
	ContactPhone    string
}

type ListOptions struct {
	Status     string
	MerchantID string
	ConsumerID string
	DemandID   string
	Limit      int
	Offset     int
}

type ReviewInput struct {
	Status     string
	ReviewerID string
	Reason     string
}

type Store interface {
	Ping(ctx context.Context) error
	Close() error

	CreateOrGetUser(ctx context.Context, input CreateUserInput) (domain.User, error)
	GetUser(ctx context.Context, id string) (domain.User, error)
	CreateOrGetMerchant(ctx context.Context, input CreateMerchantInput) (domain.Merchant, error)
	GetMerchant(ctx context.Context, id string) (domain.Merchant, error)
	ListMerchants(ctx context.Context, status string) ([]domain.Merchant, error)
	ReviewMerchant(ctx context.Context, id string, input ReviewInput) (domain.Merchant, error)

	CreateDemand(ctx context.Context, input CreateDemandInput) (domain.Demand, error)
	FindMatchingDemand(ctx context.Context, spec domain.DemandSpec) (domain.Demand, error)
	GetDemand(ctx context.Context, id string) (domain.Demand, error)
	ListDemands(ctx context.Context, options ListOptions) ([]domain.Demand, error)
	AddDemandMember(ctx context.Context, input CreateMemberInput) (domain.DemandMember, domain.Demand, error)
	GetDemandMember(ctx context.Context, demandID, userID string) (domain.DemandMember, error)
	ReviewDemand(ctx context.Context, id string, input ReviewInput) (domain.Demand, error)
	ListDemandMembers(ctx context.Context, demandID string) ([]domain.DemandMember, error)

	CreateOffer(ctx context.Context, input CreateOfferInput) (domain.Offer, error)
	GetOffer(ctx context.Context, id string) (domain.Offer, error)
	ListOffers(ctx context.Context, options ListOptions) ([]domain.Offer, error)

	CreateCampaign(ctx context.Context, input CreateCampaignInput) (domain.Campaign, error)
	GetCampaign(ctx context.Context, id string) (domain.Campaign, error)
	ListCampaigns(ctx context.Context, options ListOptions) ([]domain.Campaign, error)
	ReviewCampaign(ctx context.Context, id string, input ReviewInput) (domain.Campaign, error)

	CreateOrder(ctx context.Context, input CreateOrderInput) (domain.Order, error)
	GetOrder(ctx context.Context, id string) (domain.Order, error)
	ListOrders(ctx context.Context, options ListOptions) ([]domain.Order, error)

	GetIdempotency(ctx context.Context, actorID, key string) (domain.IdempotencyRecord, error)
	PutIdempotency(ctx context.Context, record domain.IdempotencyRecord) error
}
