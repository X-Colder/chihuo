package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/domain"
)

type devLoginRequest struct {
	Code         string      `json:"code"`
	Name         string      `json:"name"`
	Role         domain.Role `json:"role"`
	MerchantName string      `json:"merchant_name,omitempty"`
}

type createDemandRequest struct {
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
	MinimumMembers  int      `json:"minimum_members"`
	MaximumMembers  int      `json:"maximum_members"`
}

type joinDemandRequest struct {
	Quantity    int      `json:"quantity"`
	WeightGrams int      `json:"weight_grams,omitempty"`
	Preferences []string `json:"preferences"`
	Notes       string   `json:"notes,omitempty"`
}

type reviewRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type createOfferRequest struct {
	DemandID            string   `json:"demand_id"`
	UnitPriceCents      int64    `json:"unit_price_cents"`
	ProductionCapacity  int      `json:"production_capacity"`
	WeightGrams         int      `json:"weight_grams"`
	Ingredients         []string `json:"ingredients"`
	Allergens           []string `json:"allergens"`
	OilLevel            string   `json:"oil_level"`
	SaltLevel           string   `json:"salt_level"`
	ProductionTime      string   `json:"production_time"`
	ShelfLifeMinutes    int      `json:"shelf_life_minutes"`
	StorageInstructions string   `json:"storage_instructions"`
	Notes               string   `json:"notes,omitempty"`
}

type createCampaignRequest struct {
	DemandID         string      `json:"demand_id"`
	OfferID          string      `json:"offer_id"`
	Title            string      `json:"title"`
	Description      string      `json:"description,omitempty"`
	UnitPriceCents   int64       `json:"unit_price_cents"`
	DeliveryFeeCents int64       `json:"delivery_fee_cents"`
	PlatformFeeBPS   int64       `json:"platform_fee_bps"`
	MinimumOrders    int         `json:"minimum_orders"`
	MaximumOrders    int         `json:"maximum_orders"`
	StartsAt         time.Time   `json:"starts_at"`
	EndsAt           time.Time   `json:"ends_at"`
	PickupPoint      string      `json:"pickup_point"`
	FoodSpec         foodSpecDTO `json:"food_spec"`
}

type foodSpecDTO struct {
	WeightGrams         int      `json:"weight_grams"`
	Ingredients         []string `json:"ingredients"`
	Allergens           []string `json:"allergens"`
	OilLevel            string   `json:"oil_level"`
	SaltLevel           string   `json:"salt_level"`
	ProductionTime      string   `json:"production_time"`
	ShelfLifeMinutes    int      `json:"shelf_life_minutes"`
	StorageInstructions string   `json:"storage_instructions"`
}

type createOrderRequest struct {
	Quantity        int    `json:"quantity"`
	DeliveryAddress string `json:"delivery_address"`
	ContactName     string `json:"contact_name"`
	ContactPhone    string `json:"contact_phone"`
}

func (v devLoginRequest) validate() error {
	if err := requiredText(v.Code, "code", 128); err != nil {
		return err
	}
	if err := requiredText(v.Name, "name", 80); err != nil {
		return err
	}
	if v.Role == "" {
		v.Role = domain.RoleConsumer
	}
	if !v.Role.Valid() {
		return newRequestError(http.StatusBadRequest, "INVALID_ROLE", "role is invalid", nil)
	}
	if v.Role == domain.RoleMerchant && v.MerchantName != "" {
		return requiredText(v.MerchantName, "merchant_name", 120)
	}
	return nil
}

func (v createDemandRequest) toSpec() (domain.DemandSpec, error) {
	if err := requiredText(v.Category, "category", 80); err != nil {
		return domain.DemandSpec{}, err
	}
	if err := requiredText(v.Title, "title", 120); err != nil {
		return domain.DemandSpec{}, err
	}
	if err := requiredText(v.ServiceArea, "service_area", 120); err != nil {
		return domain.DemandSpec{}, err
	}
	if _, err := time.Parse("2006-01-02", v.ServingDate); err != nil {
		return domain.DemandSpec{}, newRequestError(http.StatusBadRequest, "INVALID_DATE", "serving_date must be YYYY-MM-DD", nil)
	}
	if _, err := time.Parse("15:04", v.ServingTime); err != nil {
		return domain.DemandSpec{}, newRequestError(http.StatusBadRequest, "INVALID_TIME", "serving_time must be HH:mm", nil)
	}
	if v.BudgetMinCents <= 0 || v.BudgetMaxCents < v.BudgetMinCents {
		return domain.DemandSpec{}, newRequestError(http.StatusBadRequest, "INVALID_BUDGET", "budget range is invalid", nil)
	}
	if v.Quantity <= 0 || v.WeightMinGrams <= 0 || v.WeightMaxGrams < v.WeightMinGrams {
		return domain.DemandSpec{}, newRequestError(http.StatusBadRequest, "INVALID_QUANTITY", "quantity or weight range is invalid", nil)
	}
	if v.MinimumMembers <= 0 || v.MaximumMembers < v.MinimumMembers {
		return domain.DemandSpec{}, newRequestError(http.StatusBadRequest, "INVALID_MEMBER_LIMIT", "member limits are invalid", nil)
	}
	return domain.DemandSpec{
		Category:        strings.TrimSpace(v.Category),
		Title:           strings.TrimSpace(v.Title),
		ServiceArea:     strings.TrimSpace(v.ServiceArea),
		ServingDate:     v.ServingDate,
		ServingTime:     v.ServingTime,
		BudgetMinCents:  v.BudgetMinCents,
		BudgetMaxCents:  v.BudgetMaxCents,
		Quantity:        v.Quantity,
		WeightMinGrams:  v.WeightMinGrams,
		WeightMaxGrams:  v.WeightMaxGrams,
		HardConstraints: normalizeStrings(v.HardConstraints, 20, 80),
		Preferences:     normalizeStrings(v.Preferences, 20, 80),
		Notes:           optionalText(v.Notes, 1000),
	}, nil
}

func (v joinDemandRequest) validate() error {
	if v.Quantity <= 0 {
		return newRequestError(http.StatusBadRequest, "INVALID_QUANTITY", "quantity must be positive", nil)
	}
	if v.WeightGrams < 0 {
		return newRequestError(http.StatusBadRequest, "INVALID_WEIGHT", "weight_grams must not be negative", nil)
	}
	return nil
}

func (v reviewRequest) validate(allowed ...string) error {
	for _, status := range allowed {
		if v.Status == status {
			if len(v.Reason) > 500 {
				return newRequestError(http.StatusBadRequest, "INVALID_REASON", "reason is too long", nil)
			}
			return nil
		}
	}
	return newRequestError(http.StatusBadRequest, "INVALID_STATUS", "status is not allowed", allowed)
}

func (v createOfferRequest) validate() error {
	if requiredText(v.DemandID, "demand_id", 64) != nil {
		return requiredText(v.DemandID, "demand_id", 64)
	}
	if v.UnitPriceCents <= 0 || v.ProductionCapacity <= 0 || v.WeightGrams <= 0 || v.ShelfLifeMinutes <= 0 {
		return newRequestError(http.StatusBadRequest, "INVALID_OFFER", "offer numeric fields are invalid", nil)
	}
	if err := requiredText(v.StorageInstructions, "storage_instructions", 300); err != nil {
		return err
	}
	if _, err := time.Parse("15:04", v.ProductionTime); err != nil {
		return newRequestError(http.StatusBadRequest, "INVALID_TIME", "production_time must be HH:mm", nil)
	}
	if !validFoodLevel(v.OilLevel) || !validFoodLevel(v.SaltLevel) {
		return newRequestError(http.StatusBadRequest, "INVALID_FOOD_LEVEL", "oil_level and salt_level are invalid", nil)
	}
	v.Ingredients = normalizeStrings(v.Ingredients, 100, 100)
	v.Allergens = normalizeStrings(v.Allergens, 50, 100)
	return nil
}

func (v createCampaignRequest) validate() error {
	if err := requiredText(v.DemandID, "demand_id", 64); err != nil {
		return err
	}
	if err := requiredText(v.OfferID, "offer_id", 64); err != nil {
		return err
	}
	if err := requiredText(v.Title, "title", 120); err != nil {
		return err
	}
	if v.UnitPriceCents <= 0 || v.DeliveryFeeCents < 0 || v.PlatformFeeBPS < 0 || v.PlatformFeeBPS > 2000 {
		return newRequestError(http.StatusBadRequest, "INVALID_PRICE", "campaign price fields are invalid", nil)
	}
	if v.MinimumOrders <= 0 || v.MaximumOrders < v.MinimumOrders {
		return newRequestError(http.StatusBadRequest, "INVALID_ORDER_LIMIT", "campaign order limits are invalid", nil)
	}
	if v.EndsAt.IsZero() || v.StartsAt.IsZero() || !v.EndsAt.After(v.StartsAt) {
		return newRequestError(http.StatusBadRequest, "INVALID_TIME_RANGE", "campaign time range is invalid", nil)
	}
	if err := requiredText(v.PickupPoint, "pickup_point", 200); err != nil {
		return err
	}
	if v.FoodSpec.WeightGrams <= 0 || v.FoodSpec.ShelfLifeMinutes <= 0 {
		return newRequestError(http.StatusBadRequest, "INVALID_FOOD_SPEC", "food_spec numeric fields are invalid", nil)
	}
	if _, err := time.Parse("15:04", v.FoodSpec.ProductionTime); err != nil {
		return newRequestError(http.StatusBadRequest, "INVALID_TIME", "food_spec.production_time must be HH:mm", nil)
	}
	if !validFoodLevel(v.FoodSpec.OilLevel) || !validFoodLevel(v.FoodSpec.SaltLevel) {
		return newRequestError(http.StatusBadRequest, "INVALID_FOOD_LEVEL", "food_spec oil_level and salt_level are invalid", nil)
	}
	if err := requiredText(v.FoodSpec.StorageInstructions, "food_spec.storage_instructions", 300); err != nil {
		return err
	}
	return nil
}

func validFoodLevel(value string) bool {
	switch value {
	case "", "UNKNOWN", "LOW", "MEDIUM", "HIGH":
		return true
	default:
		return false
	}
}

func (v createOrderRequest) validate() error {
	if v.Quantity <= 0 {
		return newRequestError(http.StatusBadRequest, "INVALID_QUANTITY", "quantity must be positive", nil)
	}
	if err := requiredText(v.DeliveryAddress, "delivery_address", 300); err != nil {
		return err
	}
	if err := requiredText(v.ContactName, "contact_name", 80); err != nil {
		return err
	}
	if err := requiredText(v.ContactPhone, "contact_phone", 30); err != nil {
		return err
	}
	return nil
}

func requiredText(value, field string, max int) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return newRequestError(http.StatusBadRequest, "INVALID_"+strings.ToUpper(field), field+" is required", nil)
	}
	if len([]rune(trimmed)) > max {
		return newRequestError(http.StatusBadRequest, "INVALID_"+strings.ToUpper(field), field+" is too long", nil)
	}
	return nil
}

func optionalText(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if len([]rune(trimmed)) > max {
		return string([]rune(trimmed)[:max])
	}
	return trimmed
}

func normalizeStrings(values []string, maxItems, maxLength int) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(result) >= maxItems {
			continue
		}
		if len([]rune(value)) > maxLength {
			value = string([]rune(value)[:maxLength])
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}
