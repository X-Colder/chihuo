package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/domain"
)

type MemoryStore struct {
	mu              sync.RWMutex
	users           map[string]domain.User
	usersByKey      map[string]string
	merchants       map[string]domain.Merchant
	merchantByOwner map[string]string
	demands         map[string]domain.Demand
	members         map[string]domain.DemandMember
	offers          map[string]domain.Offer
	campaigns       map[string]domain.Campaign
	orders          map[string]domain.Order
	idempotency     map[string]domain.IdempotencyRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:           make(map[string]domain.User),
		usersByKey:      make(map[string]string),
		merchants:       make(map[string]domain.Merchant),
		merchantByOwner: make(map[string]string),
		demands:         make(map[string]domain.Demand),
		members:         make(map[string]domain.DemandMember),
		offers:          make(map[string]domain.Offer),
		campaigns:       make(map[string]domain.Campaign),
		orders:          make(map[string]domain.Order),
		idempotency:     make(map[string]domain.IdempotencyRecord),
	}
}

func (s *MemoryStore) Ping(context.Context) error { return nil }
func (s *MemoryStore) Close() error               { return nil }

func (s *MemoryStore) CreateOrGetUser(_ context.Context, input CreateUserInput) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.usersByKey[input.ExternalKey]; ok {
		return cloneUser(s.users[id]), nil
	}
	now := time.Now().UTC()
	user := domain.User{ID: newID(), Name: input.Name, Role: input.Role, CreatedAt: now}
	s.users[user.ID] = user
	s.usersByKey[input.ExternalKey] = user.ID
	return cloneUser(user), nil
}

func (s *MemoryStore) GetUser(_ context.Context, id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	return cloneUser(user), nil
}

func (s *MemoryStore) CreateOrGetMerchant(_ context.Context, input CreateMerchantInput) (domain.Merchant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.merchantByOwner[input.OwnerUserID]; ok {
		return cloneMerchant(s.merchants[id]), nil
	}
	now := time.Now().UTC()
	merchant := domain.Merchant{
		ID:          newID(),
		OwnerUserID: input.OwnerUserID,
		Name:        input.Name,
		Status:      domain.MerchantPending,
		License:     cloneMap(input.License),
		CreatedAt:   now,
	}
	s.merchants[merchant.ID] = merchant
	s.merchantByOwner[merchant.OwnerUserID] = merchant.ID
	return cloneMerchant(merchant), nil
}

func (s *MemoryStore) GetMerchant(_ context.Context, id string) (domain.Merchant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	merchant, ok := s.merchants[id]
	if !ok {
		return domain.Merchant{}, ErrNotFound
	}
	return cloneMerchant(merchant), nil
}

func (s *MemoryStore) ListMerchants(_ context.Context, status string) ([]domain.Merchant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Merchant, 0)
	for _, merchant := range s.merchants {
		if status != "" && string(merchant.Status) != status {
			continue
		}
		result = append(result, cloneMerchant(merchant))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *MemoryStore) ReviewMerchant(_ context.Context, id string, input ReviewInput) (domain.Merchant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	merchant, ok := s.merchants[id]
	if !ok {
		return domain.Merchant{}, ErrNotFound
	}
	merchant.Status = domain.MerchantStatus(input.Status)
	merchant.ReviewReason = input.Reason
	now := time.Now().UTC()
	merchant.ReviewedAt = &now
	s.merchants[id] = merchant
	return cloneMerchant(merchant), nil
}

func (s *MemoryStore) CreateDemand(_ context.Context, input CreateDemandInput) (domain.Demand, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	demand := domain.Demand{
		ID:             newID(),
		CreatedBy:      input.CreatedBy,
		DemandSpec:     cloneDemandSpec(input.Spec),
		MinimumMembers: input.MinimumMembers,
		MaximumMembers: input.MaximumMembers,
		Status:         domain.DemandPendingReview,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.demands[demand.ID] = demand
	return cloneDemand(demand), nil
}

func (s *MemoryStore) FindMatchingDemand(_ context.Context, spec domain.DemandSpec) (domain.Demand, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, demand := range s.demands {
		if demand.Status != domain.DemandPendingReview && demand.Status != domain.DemandOpen {
			continue
		}
		if demand.MemberCount >= demand.MaximumMembers || !matchingSpec(demand.DemandSpec, spec) {
			continue
		}
		return cloneDemand(demand), nil
	}
	return domain.Demand{}, ErrNotFound
}

func (s *MemoryStore) GetDemand(_ context.Context, id string) (domain.Demand, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	demand, ok := s.demands[id]
	if !ok {
		return domain.Demand{}, ErrNotFound
	}
	return cloneDemand(demand), nil
}

func (s *MemoryStore) ListDemands(_ context.Context, options ListOptions) ([]domain.Demand, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Demand, 0)
	for _, demand := range s.demands {
		if options.Status != "" && string(demand.Status) != options.Status {
			continue
		}
		result = append(result, cloneDemand(demand))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return paginate(result, options.Offset, options.Limit), nil
}

func (s *MemoryStore) AddDemandMember(_ context.Context, input CreateMemberInput) (domain.DemandMember, domain.Demand, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	demand, ok := s.demands[input.DemandID]
	if !ok {
		return domain.DemandMember{}, domain.Demand{}, ErrNotFound
	}
	if demand.Status == domain.DemandRejected || demand.Status == domain.DemandClosed {
		return domain.DemandMember{}, domain.Demand{}, ErrInvalid
	}
	memberKey := input.DemandID + ":" + input.UserID
	if _, exists := s.members[memberKey]; exists {
		return domain.DemandMember{}, domain.Demand{}, ErrConflict
	}
	if demand.MemberCount >= demand.MaximumMembers {
		return domain.DemandMember{}, domain.Demand{}, ErrConflict
	}
	member := domain.DemandMember{
		ID:          newID(),
		DemandID:    input.DemandID,
		UserID:      input.UserID,
		Quantity:    input.Quantity,
		WeightGrams: input.WeightGrams,
		Preferences: cloneStrings(input.Preferences),
		Notes:       input.Notes,
		CreatedAt:   time.Now().UTC(),
	}
	s.members[memberKey] = member
	demand.MemberCount++
	if demand.MemberCount >= demand.MinimumMembers && demand.Status == domain.DemandOpen {
		demand.Status = domain.DemandReady
	}
	demand.UpdatedAt = time.Now().UTC()
	s.demands[demand.ID] = demand
	return cloneMember(member), cloneDemand(demand), nil
}

func (s *MemoryStore) GetDemandMember(_ context.Context, demandID, userID string) (domain.DemandMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	member, ok := s.members[demandID+":"+userID]
	if !ok {
		return domain.DemandMember{}, ErrNotFound
	}
	return cloneMember(member), nil
}

func (s *MemoryStore) ReviewDemand(_ context.Context, id string, input ReviewInput) (domain.Demand, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	demand, ok := s.demands[id]
	if !ok {
		return domain.Demand{}, ErrNotFound
	}
	demand.Status = domain.DemandStatus(input.Status)
	demand.ReviewedBy = input.ReviewerID
	demand.ReviewReason = input.Reason
	now := time.Now().UTC()
	demand.ReviewedAt = &now
	demand.UpdatedAt = now
	s.demands[id] = demand
	return cloneDemand(demand), nil
}

func (s *MemoryStore) ListDemandMembers(_ context.Context, demandID string) ([]domain.DemandMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.demands[demandID]; !ok {
		return nil, ErrNotFound
	}
	result := make([]domain.DemandMember, 0)
	for _, member := range s.members {
		if member.DemandID == demandID {
			result = append(result, cloneMember(member))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}

func (s *MemoryStore) CreateOffer(_ context.Context, input CreateOfferInput) (domain.Offer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	demand, ok := s.demands[input.DemandID]
	if !ok {
		return domain.Offer{}, ErrNotFound
	}
	if demand.Status == domain.DemandRejected || demand.Status == domain.DemandClosed {
		return domain.Offer{}, ErrInvalid
	}
	merchant, ok := s.merchants[input.MerchantID]
	if !ok {
		return domain.Offer{}, ErrNotFound
	}
	if merchant.Status != domain.MerchantApproved {
		return domain.Offer{}, ErrForbidden
	}
	offer := domain.Offer{
		ID:                  newID(),
		DemandID:            input.DemandID,
		MerchantID:          input.MerchantID,
		UnitPriceCents:      input.UnitPriceCents,
		ProductionCapacity:  input.ProductionCapacity,
		WeightGrams:         input.WeightGrams,
		Ingredients:         cloneStrings(input.Ingredients),
		Allergens:           cloneStrings(input.Allergens),
		OilLevel:            input.OilLevel,
		SaltLevel:           input.SaltLevel,
		ProductionTime:      input.ProductionTime,
		ShelfLifeMinutes:    input.ShelfLifeMinutes,
		StorageInstructions: input.StorageInstructions,
		Notes:               input.Notes,
		Status:              domain.OfferSubmitted,
		CreatedAt:           time.Now().UTC(),
	}
	s.offers[offer.ID] = offer
	return cloneOffer(offer), nil
}

func (s *MemoryStore) GetOffer(_ context.Context, id string) (domain.Offer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	offer, ok := s.offers[id]
	if !ok {
		return domain.Offer{}, ErrNotFound
	}
	return cloneOffer(offer), nil
}

func (s *MemoryStore) ListOffers(_ context.Context, options ListOptions) ([]domain.Offer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Offer, 0)
	for _, offer := range s.offers {
		if options.DemandID != "" && offer.DemandID != options.DemandID {
			continue
		}
		if options.MerchantID != "" && offer.MerchantID != options.MerchantID {
			continue
		}
		if options.Status != "" && string(offer.Status) != options.Status {
			continue
		}
		result = append(result, cloneOffer(offer))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return paginate(result, options.Offset, options.Limit), nil
}

func (s *MemoryStore) CreateCampaign(_ context.Context, input CreateCampaignInput) (domain.Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	demand, ok := s.demands[input.DemandID]
	if !ok {
		return domain.Campaign{}, ErrNotFound
	}
	offer, ok := s.offers[input.OfferID]
	if !ok || offer.DemandID != input.DemandID || offer.MerchantID != input.MerchantID {
		return domain.Campaign{}, ErrConflict
	}
	campaign := domain.Campaign{
		ID:               newID(),
		DemandID:         input.DemandID,
		OfferID:          input.OfferID,
		MerchantID:       input.MerchantID,
		Title:            input.Title,
		Description:      input.Description,
		UnitPriceCents:   input.UnitPriceCents,
		DeliveryFeeCents: input.DeliveryFeeCents,
		PlatformFeeBPS:   input.PlatformFeeBPS,
		MinimumOrders:    input.MinimumOrders,
		MaximumOrders:    input.MaximumOrders,
		StartsAt:         input.StartsAt,
		EndsAt:           input.EndsAt,
		PickupPoint:      input.PickupPoint,
		FoodSpec:         cloneFoodSpec(input.FoodSpec),
		Status:           domain.CampaignPendingReview,
		CreatedAt:        time.Now().UTC(),
	}
	if demand.Status == domain.DemandRejected || demand.Status == domain.DemandClosed {
		return domain.Campaign{}, ErrInvalid
	}
	campaign.UpdatedAt = campaign.CreatedAt
	s.campaigns[campaign.ID] = campaign
	return cloneCampaign(campaign), nil
}

func (s *MemoryStore) GetCampaign(_ context.Context, id string) (domain.Campaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	campaign, ok := s.campaigns[id]
	if !ok {
		return domain.Campaign{}, ErrNotFound
	}
	return cloneCampaign(campaign), nil
}

func (s *MemoryStore) ListCampaigns(_ context.Context, options ListOptions) ([]domain.Campaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Campaign, 0)
	for _, campaign := range s.campaigns {
		if options.Status != "" && string(campaign.Status) != options.Status {
			continue
		}
		if options.MerchantID != "" && campaign.MerchantID != options.MerchantID {
			continue
		}
		if options.DemandID != "" && campaign.DemandID != options.DemandID {
			continue
		}
		result = append(result, cloneCampaign(campaign))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return paginate(result, options.Offset, options.Limit), nil
}

func (s *MemoryStore) ReviewCampaign(_ context.Context, id string, input ReviewInput) (domain.Campaign, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	campaign, ok := s.campaigns[id]
	if !ok {
		return domain.Campaign{}, ErrNotFound
	}
	if input.Status == string(domain.CampaignOpen) {
		demand := s.demands[campaign.DemandID]
		if demand.Status == domain.DemandRejected || demand.Status == domain.DemandClosed {
			return domain.Campaign{}, ErrInvalid
		}
		if campaign.MaximumOrders < campaign.MinimumOrders {
			return domain.Campaign{}, ErrInvalid
		}
		offer := s.offers[campaign.OfferID]
		offer.Status = domain.OfferAccepted
		s.offers[offer.ID] = offer
	}
	campaign.Status = domain.CampaignStatus(input.Status)
	campaign.ReviewReason = input.Reason
	now := time.Now().UTC()
	campaign.ReviewedAt = &now
	campaign.UpdatedAt = now
	s.campaigns[id] = campaign
	return cloneCampaign(campaign), nil
}

func (s *MemoryStore) CreateOrder(_ context.Context, input CreateOrderInput) (domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	campaign, ok := s.campaigns[input.CampaignID]
	if !ok {
		return domain.Order{}, ErrNotFound
	}
	if campaign.Status != domain.CampaignOpen {
		return domain.Order{}, ErrInvalid
	}
	if campaign.CurrentOrders+input.Quantity > campaign.MaximumOrders {
		return domain.Order{}, ErrConflict
	}
	subtotal := campaign.UnitPriceCents * int64(input.Quantity)
	platformFee := subtotal * campaign.PlatformFeeBPS / 10_000
	now := time.Now().UTC()
	order := domain.Order{
		ID:               newID(),
		CampaignID:       input.CampaignID,
		ConsumerID:       input.ConsumerID,
		Quantity:         input.Quantity,
		DeliveryAddress:  input.DeliveryAddress,
		ContactName:      input.ContactName,
		ContactPhone:     input.ContactPhone,
		Status:           domain.OrderPendingPayment,
		UnitPriceCents:   campaign.UnitPriceCents,
		SubtotalCents:    subtotal,
		DeliveryFeeCents: campaign.DeliveryFeeCents,
		PlatformFeeCents: platformFee,
		TotalCents:       subtotal + campaign.DeliveryFeeCents + platformFee,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.orders[order.ID] = order
	campaign.CurrentOrders += input.Quantity
	if campaign.CurrentOrders >= campaign.MaximumOrders {
		campaign.Status = domain.CampaignSoldOut
	}
	campaign.UpdatedAt = now
	s.campaigns[campaign.ID] = campaign
	return cloneOrder(order), nil
}

func (s *MemoryStore) GetOrder(_ context.Context, id string) (domain.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	if !ok {
		return domain.Order{}, ErrNotFound
	}
	return cloneOrder(order), nil
}

func (s *MemoryStore) ListOrders(_ context.Context, options ListOptions) ([]domain.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Order, 0)
	for _, order := range s.orders {
		if options.ConsumerID != "" && order.ConsumerID != options.ConsumerID {
			continue
		}
		if options.Status != "" && string(order.Status) != options.Status {
			continue
		}
		result = append(result, cloneOrder(order))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return paginate(result, options.Offset, options.Limit), nil
}

func (s *MemoryStore) GetIdempotency(_ context.Context, actorID, key string) (domain.IdempotencyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.idempotency[actorID+":"+key]
	if !ok {
		return domain.IdempotencyRecord{}, ErrNotFound
	}
	record.Response = append([]byte(nil), record.Response...)
	return record, nil
}

func (s *MemoryStore) PutIdempotency(_ context.Context, record domain.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := record.ActorID + ":" + record.Key
	if existing, ok := s.idempotency[key]; ok {
		if existing.Fingerprint != record.Fingerprint {
			return ErrConflict
		}
		return nil
	}
	record.Response = append([]byte(nil), record.Response...)
	s.idempotency[key] = record
	return nil
}

func matchingSpec(a, b domain.DemandSpec) bool {
	return strings.EqualFold(a.Category, b.Category) &&
		strings.EqualFold(a.ServiceArea, b.ServiceArea) &&
		a.ServingDate == b.ServingDate &&
		a.ServingTime == b.ServingTime &&
		intervalsOverlap(a.BudgetMinCents, a.BudgetMaxCents, b.BudgetMinCents, b.BudgetMaxCents) &&
		intervalsOverlap(int64(a.WeightMinGrams), int64(a.WeightMaxGrams), int64(b.WeightMinGrams), int64(b.WeightMaxGrams)) &&
		equalStringSet(a.HardConstraints, b.HardConstraints)
}

func intervalsOverlap(aMin, aMax, bMin, bMax int64) bool {
	return aMin <= bMax && bMin <= aMax
}

func equalStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]struct{}, len(a))
	for _, value := range a {
		seen[strings.ToLower(strings.TrimSpace(value))] = struct{}{}
	}
	for _, value := range b {
		if _, ok := seen[strings.ToLower(strings.TrimSpace(value))]; !ok {
			return false
		}
	}
	return true
}

func paginate[T any](items []T, offset, limit int) []T {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset >= len(items) {
		return []T{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic("crypto/rand unavailable")
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return hex.EncodeToString(bytes[0:4]) + "-" +
		hex.EncodeToString(bytes[4:6]) + "-" +
		hex.EncodeToString(bytes[6:8]) + "-" +
		hex.EncodeToString(bytes[8:10]) + "-" +
		hex.EncodeToString(bytes[10:16])
}

func cloneUser(value domain.User) domain.User { return value }

func cloneMerchant(value domain.Merchant) domain.Merchant {
	value.License = cloneMap(value.License)
	return value
}

func cloneDemand(value domain.Demand) domain.Demand {
	value.DemandSpec = cloneDemandSpec(value.DemandSpec)
	return value
}

func cloneDemandSpec(value domain.DemandSpec) domain.DemandSpec {
	value.HardConstraints = cloneStrings(value.HardConstraints)
	value.Preferences = cloneStrings(value.Preferences)
	return value
}

func cloneMember(value domain.DemandMember) domain.DemandMember {
	value.Preferences = cloneStrings(value.Preferences)
	return value
}

func cloneOffer(value domain.Offer) domain.Offer {
	value.Ingredients = cloneStrings(value.Ingredients)
	value.Allergens = cloneStrings(value.Allergens)
	return value
}

func cloneFoodSpec(value domain.FoodSpec) domain.FoodSpec {
	value.Ingredients = cloneStrings(value.Ingredients)
	value.Allergens = cloneStrings(value.Allergens)
	return value
}

func cloneCampaign(value domain.Campaign) domain.Campaign {
	value.FoodSpec = cloneFoodSpec(value.FoodSpec)
	return value
}

func cloneOrder(value domain.Order) domain.Order { return value }

func cloneStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	return mapsClone(values)
}

func mapsClone(values map[string]any) map[string]any {
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
