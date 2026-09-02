package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStore is the production persistence implementation. Business
// operations use transactions where a read-modify-write sequence must be
// atomic.
type PostgresStore struct {
	db *sql.DB
}

var _ Store = (*PostgresStore)(nil)

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	if s == nil || s.db == nil {
		return ErrInvalid
	}
	return s.db.PingContext(ctx)
}

func (s *PostgresStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

//go:embed migrations/001_initial.sql
var initialMigration string

func (s *PostgresStore) Migrate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, initialMigration); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) CreateOrGetUser(ctx context.Context, input CreateUserInput) (domain.User, error) {
	id, err := newUUID()
	if err != nil {
		return domain.User{}, err
	}
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO users (id, name, role, external_key, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (external_key) DO UPDATE SET external_key = EXCLUDED.external_key
		RETURNING id, name, role, created_at
	`, id, input.Name, input.Role, input.ExternalKey, now)
	user, err := scanUser(row)
	return user, normalizeDBError(err)
}

func (s *PostgresStore) GetUser(ctx context.Context, id string) (domain.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, role, created_at
		FROM users
		WHERE id = $1
	`, id)
	user, err := scanUser(row)
	return user, normalizeDBError(err)
}

func (s *PostgresStore) CreateOrGetMerchant(ctx context.Context, input CreateMerchantInput) (domain.Merchant, error) {
	id, err := newUUID()
	if err != nil {
		return domain.Merchant{}, err
	}
	license, err := marshalJSON(input.License, map[string]any{})
	if err != nil {
		return domain.Merchant{}, err
	}
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO merchants (id, owner_user_id, name, status, license, created_at)
		VALUES ($1, $2, $3, 'PENDING', $4::jsonb, $5)
		ON CONFLICT (owner_user_id) DO UPDATE SET owner_user_id = EXCLUDED.owner_user_id
		RETURNING id, owner_user_id, name, status, license, review_reason, reviewed_at, created_at
	`, id, input.OwnerUserID, input.Name, license, now)
	merchant, err := scanMerchant(row)
	return merchant, normalizeDBError(err)
}

func (s *PostgresStore) GetMerchant(ctx context.Context, id string) (domain.Merchant, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, owner_user_id, name, status, license, review_reason, reviewed_at, created_at
		FROM merchants
		WHERE id = $1
	`, id)
	merchant, err := scanMerchant(row)
	return merchant, normalizeDBError(err)
}

func (s *PostgresStore) ListMerchants(ctx context.Context, status string) ([]domain.Merchant, error) {
	query := `
		SELECT id, owner_user_id, name, status, license, review_reason, reviewed_at, created_at
		FROM merchants
	`
	args := make([]any, 0, 1)
	if status != "" {
		query += " WHERE status = $1"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeDBError(err)
	}
	defer rows.Close()
	result := make([]domain.Merchant, 0)
	for rows.Next() {
		merchant, err := scanMerchant(rows)
		if err != nil {
			return nil, normalizeDBError(err)
		}
		result = append(result, merchant)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeDBError(err)
	}
	return result, nil
}

func (s *PostgresStore) ReviewMerchant(ctx context.Context, id string, input ReviewInput) (domain.Merchant, error) {
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, `
		UPDATE merchants
		SET status = $2, review_reason = $3, reviewed_at = $4
		WHERE id = $1
		RETURNING id, owner_user_id, name, status, license, review_reason, reviewed_at, created_at
	`, id, input.Status, nullableString(input.Reason), now)
	merchant, err := scanMerchant(row)
	return merchant, normalizeDBError(err)
}

func (s *PostgresStore) CreateDemand(ctx context.Context, input CreateDemandInput) (domain.Demand, error) {
	id, err := newUUID()
	if err != nil {
		return domain.Demand{}, err
	}
	hardConstraints, err := marshalJSON(input.Spec.HardConstraints, []string{})
	if err != nil {
		return domain.Demand{}, err
	}
	preferences, err := marshalJSON(input.Spec.Preferences, []string{})
	if err != nil {
		return domain.Demand{}, err
	}
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO demand_clusters (
			id, created_by, category, title, service_area, serving_date, serving_time,
			budget_min_cents, budget_max_cents, quantity, weight_min_grams, weight_max_grams,
			hard_constraints, preferences, notes, minimum_members, maximum_members,
			status, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6::date, $7, $8, $9, $10, $11, $12,
			$13::jsonb, $14::jsonb, $15, $16, $17, 'PENDING_REVIEW', $18, $18
		)
		RETURNING id, created_by, category, title, service_area, serving_date::text,
			serving_time, budget_min_cents, budget_max_cents, quantity,
			weight_min_grams, weight_max_grams, hard_constraints, preferences, notes,
			minimum_members, maximum_members, member_count, status, reviewed_by,
			reviewed_at, review_reason, created_at, updated_at
	`, id, input.CreatedBy, input.Spec.Category, input.Spec.Title, input.Spec.ServiceArea,
		input.Spec.ServingDate, input.Spec.ServingTime, input.Spec.BudgetMinCents,
		input.Spec.BudgetMaxCents, input.Spec.Quantity, input.Spec.WeightMinGrams,
		input.Spec.WeightMaxGrams, hardConstraints, preferences, nullableString(input.Spec.Notes),
		input.MinimumMembers, input.MaximumMembers, now)
	demand, err := scanDemand(row)
	return demand, normalizeDBError(err)
}

func (s *PostgresStore) FindMatchingDemand(ctx context.Context, spec domain.DemandSpec) (domain.Demand, error) {
	hardConstraints, err := marshalJSON(spec.HardConstraints, []string{})
	if err != nil {
		return domain.Demand{}, err
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT id, created_by, category, title, service_area, serving_date::text,
			serving_time, budget_min_cents, budget_max_cents, quantity,
			weight_min_grams, weight_max_grams, hard_constraints, preferences, notes,
			minimum_members, maximum_members, member_count, status, reviewed_by,
			reviewed_at, review_reason, created_at, updated_at
		FROM demand_clusters
		WHERE status IN ('PENDING_REVIEW', 'OPEN')
		  AND member_count < maximum_members
		  AND lower(category) = lower($1)
		  AND lower(service_area) = lower($2)
		  AND serving_date = $3::date
		  AND serving_time = $4
		  AND budget_min_cents <= $6
		  AND $5 <= budget_max_cents
		  AND weight_min_grams <= $8
		  AND $7 <= weight_max_grams
		  AND hard_constraints @> $9::jsonb
		  AND $9::jsonb @> hard_constraints
		ORDER BY created_at ASC
		LIMIT 1
	`, spec.Category, spec.ServiceArea, spec.ServingDate, spec.ServingTime,
		spec.BudgetMinCents, spec.BudgetMaxCents, spec.WeightMinGrams, spec.WeightMaxGrams,
		hardConstraints)
	demand, err := scanDemand(row)
	return demand, normalizeDBError(err)
}

func (s *PostgresStore) GetDemand(ctx context.Context, id string) (domain.Demand, error) {
	row := s.db.QueryRowContext(ctx, demandSelect+` WHERE id = $1`, id)
	demand, err := scanDemand(row)
	return demand, normalizeDBError(err)
}

func (s *PostgresStore) ListDemands(ctx context.Context, options ListOptions) ([]domain.Demand, error) {
	limit, offset := pagination(options)
	query := demandSelect
	conditions := make([]string, 0, 1)
	args := make([]any, 0, 3)
	if options.Status != "" {
		args = append(args, options.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeDBError(err)
	}
	defer rows.Close()
	result := make([]domain.Demand, 0)
	for rows.Next() {
		demand, err := scanDemand(rows)
		if err != nil {
			return nil, normalizeDBError(err)
		}
		result = append(result, demand)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeDBError(err)
	}
	return result, nil
}

func (s *PostgresStore) AddDemandMember(ctx context.Context, input CreateMemberInput) (domain.DemandMember, domain.Demand, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.DemandMember{}, domain.Demand{}, err
	}
	defer func() { _ = tx.Rollback() }()

	demandRow := tx.QueryRowContext(ctx, demandSelect+` WHERE id = $1 FOR UPDATE`, input.DemandID)
	demand, err := scanDemand(demandRow)
	if err != nil {
		return domain.DemandMember{}, domain.Demand{}, normalizeDBError(err)
	}
	if demand.Status == domain.DemandRejected || demand.Status == domain.DemandClosed {
		return domain.DemandMember{}, domain.Demand{}, ErrInvalid
	}
	if demand.MemberCount >= demand.MaximumMembers {
		return domain.DemandMember{}, domain.Demand{}, ErrConflict
	}

	memberID, err := newUUID()
	if err != nil {
		return domain.DemandMember{}, domain.Demand{}, err
	}
	preferences, err := marshalJSON(input.Preferences, []string{})
	if err != nil {
		return domain.DemandMember{}, domain.Demand{}, err
	}
	now := time.Now().UTC()
	var member domain.DemandMember
	var weight sql.NullInt64
	var memberPreferences []byte
	var notes sql.NullString
	err = tx.QueryRowContext(ctx, `
		INSERT INTO demand_members (
			id, demand_id, user_id, quantity, weight_grams, preferences, notes, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
		RETURNING id, demand_id, user_id, quantity, weight_grams, preferences, notes, created_at
	`, memberID, input.DemandID, input.UserID, input.Quantity, nullableInt(input.WeightGrams),
		preferences, nullableString(input.Notes), now).Scan(
		&member.ID, &member.DemandID, &member.UserID, &member.Quantity, &weight,
		&memberPreferences, &notes, &member.CreatedAt,
	)
	if err != nil {
		return domain.DemandMember{}, domain.Demand{}, normalizeDBError(err)
	}
	if weight.Valid {
		member.WeightGrams = int(weight.Int64)
	}
	if err := unmarshalJSON(memberPreferences, &member.Preferences); err != nil {
		return domain.DemandMember{}, domain.Demand{}, err
	}
	member.Notes = notes.String

	nextCount := demand.MemberCount + 1
	nextStatus := string(demand.Status)
	if nextCount >= demand.MinimumMembers && demand.Status == domain.DemandOpen {
		nextStatus = string(domain.DemandReady)
	}
	updatedRow := tx.QueryRowContext(ctx, `
		UPDATE demand_clusters
		SET member_count = $2, status = $3, updated_at = $4
		WHERE id = $1
		RETURNING id, created_by, category, title, service_area, serving_date::text,
			serving_time, budget_min_cents, budget_max_cents, quantity,
			weight_min_grams, weight_max_grams, hard_constraints, preferences, notes,
			minimum_members, maximum_members, member_count, status, reviewed_by,
			reviewed_at, review_reason, created_at, updated_at
	`, input.DemandID, nextCount, nextStatus, now)
	updated, err := scanDemand(updatedRow)
	if err != nil {
		return domain.DemandMember{}, domain.Demand{}, normalizeDBError(err)
	}
	if err := tx.Commit(); err != nil {
		return domain.DemandMember{}, domain.Demand{}, err
	}
	return member, updated, nil
}

func (s *PostgresStore) GetDemandMember(ctx context.Context, demandID, userID string) (domain.DemandMember, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, demand_id, user_id, quantity, weight_grams, preferences, notes, created_at
		FROM demand_members
		WHERE demand_id = $1 AND user_id = $2
	`, demandID, userID)
	member, err := scanMember(row)
	return member, normalizeDBError(err)
}

func (s *PostgresStore) ReviewDemand(ctx context.Context, id string, input ReviewInput) (domain.Demand, error) {
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, `
		UPDATE demand_clusters
		SET status = $2, reviewed_by = $3, reviewed_at = $4, review_reason = $5, updated_at = $4
		WHERE id = $1
		RETURNING id, created_by, category, title, service_area, serving_date::text,
			serving_time, budget_min_cents, budget_max_cents, quantity,
			weight_min_grams, weight_max_grams, hard_constraints, preferences, notes,
			minimum_members, maximum_members, member_count, status, reviewed_by,
			reviewed_at, review_reason, created_at, updated_at
	`, id, input.Status, input.ReviewerID, now, nullableString(input.Reason))
	demand, err := scanDemand(row)
	return demand, normalizeDBError(err)
}

func (s *PostgresStore) ListDemandMembers(ctx context.Context, demandID string) ([]domain.DemandMember, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM demand_clusters WHERE id = $1)`, demandID).Scan(&exists); err != nil {
		return nil, normalizeDBError(err)
	}
	if !exists {
		return nil, ErrNotFound
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, demand_id, user_id, quantity, weight_grams, preferences, notes, created_at
		FROM demand_members
		WHERE demand_id = $1
		ORDER BY created_at ASC
	`, demandID)
	if err != nil {
		return nil, normalizeDBError(err)
	}
	defer rows.Close()
	result := make([]domain.DemandMember, 0)
	for rows.Next() {
		member, err := scanMember(rows)
		if err != nil {
			return nil, normalizeDBError(err)
		}
		result = append(result, member)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeDBError(err)
	}
	return result, nil
}

func (s *PostgresStore) CreateOffer(ctx context.Context, input CreateOfferInput) (domain.Offer, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Offer{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var demandStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM demand_clusters WHERE id = $1`, input.DemandID).Scan(&demandStatus); err != nil {
		return domain.Offer{}, normalizeDBError(err)
	}
	if demandStatus == string(domain.DemandRejected) || demandStatus == string(domain.DemandClosed) {
		return domain.Offer{}, ErrInvalid
	}
	var merchantStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM merchants WHERE id = $1`, input.MerchantID).Scan(&merchantStatus); err != nil {
		return domain.Offer{}, normalizeDBError(err)
	}
	if merchantStatus != string(domain.MerchantApproved) {
		return domain.Offer{}, ErrForbidden
	}
	id, err := newUUID()
	if err != nil {
		return domain.Offer{}, err
	}
	ingredients, err := marshalJSON(input.Ingredients, []string{})
	if err != nil {
		return domain.Offer{}, err
	}
	allergens, err := marshalJSON(input.Allergens, []string{})
	if err != nil {
		return domain.Offer{}, err
	}
	now := time.Now().UTC()
	row := tx.QueryRowContext(ctx, `
		INSERT INTO offers (
			id, demand_id, merchant_id, unit_price_cents, production_capacity, weight_grams,
			ingredients, allergens, oil_level, salt_level, production_time,
			shelf_life_minutes, storage_instructions, notes, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9, $10, $11, $12, $13, $14,
			'SUBMITTED', $15)
		RETURNING id, demand_id, merchant_id, unit_price_cents, production_capacity, weight_grams,
			ingredients, allergens, oil_level, salt_level, production_time,
			shelf_life_minutes, storage_instructions, notes, status, created_at
	`, id, input.DemandID, input.MerchantID, input.UnitPriceCents, input.ProductionCapacity,
		input.WeightGrams, ingredients, allergens, input.OilLevel, input.SaltLevel,
		input.ProductionTime, input.ShelfLifeMinutes, input.StorageInstructions,
		nullableString(input.Notes), now)
	offer, err := scanOffer(row)
	if err != nil {
		return domain.Offer{}, normalizeDBError(err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Offer{}, err
	}
	return offer, nil
}

func (s *PostgresStore) GetOffer(ctx context.Context, id string) (domain.Offer, error) {
	row := s.db.QueryRowContext(ctx, offerSelect+` WHERE id = $1`, id)
	offer, err := scanOffer(row)
	return offer, normalizeDBError(err)
}

func (s *PostgresStore) ListOffers(ctx context.Context, options ListOptions) ([]domain.Offer, error) {
	limit, offset := pagination(options)
	query := offerSelect
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	if options.DemandID != "" {
		args = append(args, options.DemandID)
		conditions = append(conditions, fmt.Sprintf("demand_id = $%d", len(args)))
	}
	if options.MerchantID != "" {
		args = append(args, options.MerchantID)
		conditions = append(conditions, fmt.Sprintf("merchant_id = $%d", len(args)))
	}
	if options.Status != "" {
		args = append(args, options.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeDBError(err)
	}
	defer rows.Close()
	result := make([]domain.Offer, 0)
	for rows.Next() {
		offer, err := scanOffer(rows)
		if err != nil {
			return nil, normalizeDBError(err)
		}
		result = append(result, offer)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeDBError(err)
	}
	return result, nil
}

func (s *PostgresStore) CreateCampaign(ctx context.Context, input CreateCampaignInput) (domain.Campaign, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Campaign{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var demandStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM demand_clusters WHERE id = $1`, input.DemandID).Scan(&demandStatus); err != nil {
		return domain.Campaign{}, normalizeDBError(err)
	}
	if demandStatus == string(domain.DemandRejected) || demandStatus == string(domain.DemandClosed) {
		return domain.Campaign{}, ErrInvalid
	}
	var offerDemandID, offerMerchantID string
	if err := tx.QueryRowContext(ctx, `SELECT demand_id, merchant_id FROM offers WHERE id = $1`, input.OfferID).Scan(&offerDemandID, &offerMerchantID); err != nil {
		return domain.Campaign{}, normalizeDBError(err)
	}
	if offerDemandID != input.DemandID || offerMerchantID != input.MerchantID {
		return domain.Campaign{}, ErrConflict
	}
	id, err := newUUID()
	if err != nil {
		return domain.Campaign{}, err
	}
	foodSpec, err := marshalJSON(input.FoodSpec, domain.FoodSpec{Ingredients: []string{}, Allergens: []string{}})
	if err != nil {
		return domain.Campaign{}, err
	}
	now := time.Now().UTC()
	row := tx.QueryRowContext(ctx, `
		INSERT INTO campaigns (
			id, demand_id, offer_id, merchant_id, title, description, unit_price_cents,
			delivery_fee_cents, platform_fee_bps, minimum_orders, maximum_orders,
			starts_at, ends_at, pickup_point, food_spec, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb,
			'PENDING_REVIEW', $16, $16)
		RETURNING id, demand_id, offer_id, merchant_id, title, description,
			unit_price_cents, delivery_fee_cents, platform_fee_bps, minimum_orders,
			maximum_orders, current_orders, starts_at, ends_at, pickup_point, food_spec,
			status, review_reason, reviewed_at, created_at, updated_at
	`, id, input.DemandID, input.OfferID, input.MerchantID, input.Title,
		nullableString(input.Description), input.UnitPriceCents, input.DeliveryFeeCents,
		input.PlatformFeeBPS, input.MinimumOrders, input.MaximumOrders, input.StartsAt,
		input.EndsAt, input.PickupPoint, foodSpec, now)
	campaign, err := scanCampaign(row)
	if err != nil {
		return domain.Campaign{}, normalizeDBError(err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Campaign{}, err
	}
	return campaign, nil
}

func (s *PostgresStore) GetCampaign(ctx context.Context, id string) (domain.Campaign, error) {
	row := s.db.QueryRowContext(ctx, campaignSelect+` WHERE id = $1`, id)
	campaign, err := scanCampaign(row)
	return campaign, normalizeDBError(err)
}

func (s *PostgresStore) ListCampaigns(ctx context.Context, options ListOptions) ([]domain.Campaign, error) {
	limit, offset := pagination(options)
	query := campaignSelect
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	if options.Status != "" {
		args = append(args, options.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if options.MerchantID != "" {
		args = append(args, options.MerchantID)
		conditions = append(conditions, fmt.Sprintf("merchant_id = $%d", len(args)))
	}
	if options.DemandID != "" {
		args = append(args, options.DemandID)
		conditions = append(conditions, fmt.Sprintf("demand_id = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeDBError(err)
	}
	defer rows.Close()
	result := make([]domain.Campaign, 0)
	for rows.Next() {
		campaign, err := scanCampaign(rows)
		if err != nil {
			return nil, normalizeDBError(err)
		}
		result = append(result, campaign)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeDBError(err)
	}
	return result, nil
}

func (s *PostgresStore) ReviewCampaign(ctx context.Context, id string, input ReviewInput) (domain.Campaign, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Campaign{}, err
	}
	defer func() { _ = tx.Rollback() }()

	campaignRow := tx.QueryRowContext(ctx, campaignSelect+` WHERE id = $1 FOR UPDATE`, id)
	campaign, err := scanCampaign(campaignRow)
	if err != nil {
		return domain.Campaign{}, normalizeDBError(err)
	}
	if input.Status == string(domain.CampaignOpen) {
		var demandStatus string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM demand_clusters WHERE id = $1`, campaign.DemandID).Scan(&demandStatus); err != nil {
			return domain.Campaign{}, normalizeDBError(err)
		}
		if demandStatus == string(domain.DemandRejected) || demandStatus == string(domain.DemandClosed) {
			return domain.Campaign{}, ErrInvalid
		}
		if campaign.MaximumOrders < campaign.MinimumOrders {
			return domain.Campaign{}, ErrInvalid
		}
		if _, err := tx.ExecContext(ctx, `UPDATE offers SET status = 'ACCEPTED' WHERE id = $1`, campaign.OfferID); err != nil {
			return domain.Campaign{}, normalizeDBError(err)
		}
	}
	now := time.Now().UTC()
	row := tx.QueryRowContext(ctx, `
		UPDATE campaigns
		SET status = $2, review_reason = $3, reviewed_at = $4, updated_at = $4
		WHERE id = $1
		RETURNING id, demand_id, offer_id, merchant_id, title, description,
			unit_price_cents, delivery_fee_cents, platform_fee_bps, minimum_orders,
			maximum_orders, current_orders, starts_at, ends_at, pickup_point, food_spec,
			status, review_reason, reviewed_at, created_at, updated_at
	`, id, input.Status, nullableString(input.Reason), now)
	campaign, err = scanCampaign(row)
	if err != nil {
		return domain.Campaign{}, normalizeDBError(err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Campaign{}, err
	}
	return campaign, nil
}

func (s *PostgresStore) CreateOrder(ctx context.Context, input CreateOrderInput) (domain.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Order{}, err
	}
	defer func() { _ = tx.Rollback() }()

	campaignRow := tx.QueryRowContext(ctx, campaignSelect+` WHERE id = $1 FOR UPDATE`, input.CampaignID)
	campaign, err := scanCampaign(campaignRow)
	if err != nil {
		return domain.Order{}, normalizeDBError(err)
	}
	if campaign.Status != domain.CampaignOpen {
		return domain.Order{}, ErrInvalid
	}
	if input.Quantity <= 0 || campaign.CurrentOrders+input.Quantity > campaign.MaximumOrders {
		return domain.Order{}, ErrConflict
	}
	subtotal := campaign.UnitPriceCents * int64(input.Quantity)
	platformFee := subtotal * campaign.PlatformFeeBPS / 10_000
	now := time.Now().UTC()
	orderID, err := newUUID()
	if err != nil {
		return domain.Order{}, err
	}
	row := tx.QueryRowContext(ctx, `
		INSERT INTO orders (
			id, campaign_id, consumer_id, quantity, delivery_address, contact_name,
			contact_phone, status, unit_price_cents, subtotal_cents, delivery_fee_cents,
			platform_fee_cents, total_cents, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING_PAYMENT', $8, $9, $10, $11, $12, $13, $13)
		RETURNING id, campaign_id, consumer_id, quantity, delivery_address, contact_name,
			contact_phone, status, unit_price_cents, subtotal_cents, delivery_fee_cents,
			platform_fee_cents, total_cents, created_at, updated_at
	`, orderID, input.CampaignID, input.ConsumerID, input.Quantity, input.DeliveryAddress,
		input.ContactName, input.ContactPhone, campaign.UnitPriceCents, subtotal,
		campaign.DeliveryFeeCents, platformFee, subtotal+campaign.DeliveryFeeCents+platformFee, now)
	order, err := scanOrder(row)
	if err != nil {
		return domain.Order{}, normalizeDBError(err)
	}
	nextCount := campaign.CurrentOrders + input.Quantity
	nextStatus := string(campaign.Status)
	if nextCount >= campaign.MaximumOrders {
		nextStatus = string(domain.CampaignSoldOut)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE campaigns
		SET current_orders = $2, status = $3, updated_at = $4
		WHERE id = $1
	`, input.CampaignID, nextCount, nextStatus, now); err != nil {
		return domain.Order{}, normalizeDBError(err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (s *PostgresStore) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	row := s.db.QueryRowContext(ctx, orderSelect+` WHERE id = $1`, id)
	order, err := scanOrder(row)
	return order, normalizeDBError(err)
}

func (s *PostgresStore) ListOrders(ctx context.Context, options ListOptions) ([]domain.Order, error) {
	limit, offset := pagination(options)
	query := orderSelect
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if options.ConsumerID != "" {
		args = append(args, options.ConsumerID)
		conditions = append(conditions, fmt.Sprintf("consumer_id = $%d", len(args)))
	}
	if options.Status != "" {
		args = append(args, options.Status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeDBError(err)
	}
	defer rows.Close()
	result := make([]domain.Order, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, normalizeDBError(err)
		}
		result = append(result, order)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeDBError(err)
	}
	return result, nil
}

func (s *PostgresStore) GetIdempotency(ctx context.Context, actorID, key string) (domain.IdempotencyRecord, error) {
	var record domain.IdempotencyRecord
	err := s.db.QueryRowContext(ctx, `
		SELECT actor_id, idempotency_key, fingerprint, status, response, created_at
		FROM idempotency_records
		WHERE actor_id = $1 AND idempotency_key = $2
	`, actorID, key).Scan(
		&record.ActorID, &record.Key, &record.Fingerprint, &record.Status, &record.Response, &record.CreatedAt,
	)
	if err != nil {
		return domain.IdempotencyRecord{}, normalizeDBError(err)
	}
	record.Response = append([]byte(nil), record.Response...)
	return record, nil
}

func (s *PostgresStore) PutIdempotency(ctx context.Context, record domain.IdempotencyRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	response := record.Response
	if len(response) == 0 {
		response = []byte("null")
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO idempotency_records (actor_id, idempotency_key, fingerprint, status, response, created_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		ON CONFLICT (actor_id, idempotency_key) DO NOTHING
	`, record.ActorID, record.Key, record.Fingerprint, record.Status, response, record.CreatedAt); err != nil {
		return normalizeDBError(err)
	}
	var existingFingerprint string
	if err := tx.QueryRowContext(ctx, `
		SELECT fingerprint
		FROM idempotency_records
		WHERE actor_id = $1 AND idempotency_key = $2
	`, record.ActorID, record.Key).Scan(&existingFingerprint); err != nil {
		return normalizeDBError(err)
	}
	if existingFingerprint != record.Fingerprint {
		return ErrConflict
	}
	return tx.Commit()
}

const demandSelect = `
	SELECT id, created_by, category, title, service_area, serving_date::text,
		serving_time, budget_min_cents, budget_max_cents, quantity,
		weight_min_grams, weight_max_grams, hard_constraints, preferences, notes,
		minimum_members, maximum_members, member_count, status, reviewed_by,
		reviewed_at, review_reason, created_at, updated_at
		FROM demand_clusters`

const offerSelect = `
	SELECT id, demand_id, merchant_id, unit_price_cents, production_capacity, weight_grams,
		ingredients, allergens, oil_level, salt_level, production_time,
		shelf_life_minutes, storage_instructions, notes, status, created_at
		FROM offers`

const campaignSelect = `
	SELECT id, demand_id, offer_id, merchant_id, title, description,
		unit_price_cents, delivery_fee_cents, platform_fee_bps, minimum_orders,
		maximum_orders, current_orders, starts_at, ends_at, pickup_point, food_spec,
		status, review_reason, reviewed_at, created_at, updated_at
		FROM campaigns`

const orderSelect = `
	SELECT id, campaign_id, consumer_id, quantity, delivery_address, contact_name,
		contact_phone, status, unit_price_cents, subtotal_cents, delivery_fee_cents,
		platform_fee_cents, total_cents, created_at, updated_at
		FROM orders`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (domain.User, error) {
	var user domain.User
	err := row.Scan(&user.ID, &user.Name, &user.Role, &user.CreatedAt)
	return user, err
}

func scanMerchant(row rowScanner) (domain.Merchant, error) {
	var merchant domain.Merchant
	var license []byte
	var reviewReason sql.NullString
	var reviewedAt sql.NullTime
	err := row.Scan(&merchant.ID, &merchant.OwnerUserID, &merchant.Name, &merchant.Status,
		&license, &reviewReason, &reviewedAt, &merchant.CreatedAt)
	if err != nil {
		return domain.Merchant{}, err
	}
	if err := unmarshalJSON(license, &merchant.License); err != nil {
		return domain.Merchant{}, err
	}
	merchant.ReviewReason = reviewReason.String
	merchant.ReviewedAt = nullTimePtr(reviewedAt)
	return merchant, nil
}

func scanDemand(row rowScanner) (domain.Demand, error) {
	var demand domain.Demand
	var hardConstraints, preferences []byte
	var notes, reviewedBy, reviewReason sql.NullString
	var reviewedAt sql.NullTime
	err := row.Scan(&demand.ID, &demand.CreatedBy, &demand.Category, &demand.Title,
		&demand.ServiceArea, &demand.ServingDate, &demand.ServingTime,
		&demand.BudgetMinCents, &demand.BudgetMaxCents, &demand.Quantity,
		&demand.WeightMinGrams, &demand.WeightMaxGrams, &hardConstraints, &preferences,
		&notes, &demand.MinimumMembers, &demand.MaximumMembers, &demand.MemberCount,
		&demand.Status, &reviewedBy, &reviewedAt, &reviewReason, &demand.CreatedAt, &demand.UpdatedAt)
	if err != nil {
		return domain.Demand{}, err
	}
	if err := unmarshalJSON(hardConstraints, &demand.HardConstraints); err != nil {
		return domain.Demand{}, err
	}
	if err := unmarshalJSON(preferences, &demand.Preferences); err != nil {
		return domain.Demand{}, err
	}
	demand.Notes = notes.String
	demand.ReviewedBy = reviewedBy.String
	demand.ReviewedAt = nullTimePtr(reviewedAt)
	demand.ReviewReason = reviewReason.String
	return demand, nil
}

func scanMember(row rowScanner) (domain.DemandMember, error) {
	var member domain.DemandMember
	var weight sql.NullInt64
	var preferences []byte
	var notes sql.NullString
	err := row.Scan(&member.ID, &member.DemandID, &member.UserID, &member.Quantity,
		&weight, &preferences, &notes, &member.CreatedAt)
	if err != nil {
		return domain.DemandMember{}, err
	}
	if weight.Valid {
		member.WeightGrams = int(weight.Int64)
	}
	if err := unmarshalJSON(preferences, &member.Preferences); err != nil {
		return domain.DemandMember{}, err
	}
	member.Notes = notes.String
	return member, nil
}

func scanOffer(row rowScanner) (domain.Offer, error) {
	var offer domain.Offer
	var ingredients, allergens []byte
	var notes sql.NullString
	err := row.Scan(&offer.ID, &offer.DemandID, &offer.MerchantID, &offer.UnitPriceCents,
		&offer.ProductionCapacity, &offer.WeightGrams, &ingredients, &allergens,
		&offer.OilLevel, &offer.SaltLevel, &offer.ProductionTime, &offer.ShelfLifeMinutes,
		&offer.StorageInstructions, &notes, &offer.Status, &offer.CreatedAt)
	if err != nil {
		return domain.Offer{}, err
	}
	if err := unmarshalJSON(ingredients, &offer.Ingredients); err != nil {
		return domain.Offer{}, err
	}
	if err := unmarshalJSON(allergens, &offer.Allergens); err != nil {
		return domain.Offer{}, err
	}
	offer.Notes = notes.String
	return offer, nil
}

func scanCampaign(row rowScanner) (domain.Campaign, error) {
	var campaign domain.Campaign
	var description, reviewReason sql.NullString
	var reviewedAt sql.NullTime
	var foodSpec []byte
	err := row.Scan(&campaign.ID, &campaign.DemandID, &campaign.OfferID, &campaign.MerchantID,
		&campaign.Title, &description, &campaign.UnitPriceCents, &campaign.DeliveryFeeCents,
		&campaign.PlatformFeeBPS, &campaign.MinimumOrders, &campaign.MaximumOrders,
		&campaign.CurrentOrders, &campaign.StartsAt, &campaign.EndsAt, &campaign.PickupPoint,
		&foodSpec, &campaign.Status, &reviewReason, &reviewedAt, &campaign.CreatedAt, &campaign.UpdatedAt)
	if err != nil {
		return domain.Campaign{}, err
	}
	if err := unmarshalJSON(foodSpec, &campaign.FoodSpec); err != nil {
		return domain.Campaign{}, err
	}
	campaign.Description = description.String
	campaign.ReviewReason = reviewReason.String
	campaign.ReviewedAt = nullTimePtr(reviewedAt)
	return campaign, nil
}

func scanOrder(row rowScanner) (domain.Order, error) {
	var order domain.Order
	err := row.Scan(&order.ID, &order.CampaignID, &order.ConsumerID, &order.Quantity,
		&order.DeliveryAddress, &order.ContactName, &order.ContactPhone, &order.Status,
		&order.UnitPriceCents, &order.SubtotalCents, &order.DeliveryFeeCents,
		&order.PlatformFeeCents, &order.TotalCents, &order.CreatedAt, &order.UpdatedAt)
	return order, err
}

func pagination(options ListOptions) (int, int) {
	limit := options.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := options.Offset
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func marshalJSON(value any, fallback any) ([]byte, error) {
	if value == nil {
		value = fallback
	}
	return json.Marshal(value)
}

func unmarshalJSON(data []byte, target any) error {
	if len(data) == 0 {
		data = []byte("null")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode postgres jsonb: %w", err)
	}
	switch value := target.(type) {
	case *[]string:
		if *value == nil {
			*value = []string{}
		}
	case *map[string]any:
		if *value == nil {
			*value = map[string]any{}
		}
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func normalizeDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrNotFound
		case "22P02", "23514":
			return ErrInvalid
		}
	}
	return err
}

func newUUID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(buffer)
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32]), nil
}
