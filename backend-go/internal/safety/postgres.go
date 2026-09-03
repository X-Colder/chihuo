package safety

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var errSafetyDatabaseUnavailable = errors.New("safety postgres service database is unavailable")

// PostgresService persists the safety core using the schema from migration
// 002_production_p0.sql. It intentionally does not own the *sql.DB lifecycle.
type PostgresService struct {
	db  *sql.DB
	now func() time.Time
}

var _ Service = (*PostgresService)(nil)

func NewPostgresService(db *sql.DB) *PostgresService {
	return NewPostgresServiceWithClock(db, nil)
}

func NewPostgresServiceWithClock(db *sql.DB, now func() time.Time) *PostgresService {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &PostgresService{db: db, now: now}
}

func (s *PostgresService) currentTime() time.Time {
	return s.now().UTC()
}

func (s *PostgresService) ready() error {
	if s == nil || s.db == nil {
		return errSafetyDatabaseUnavailable
	}
	return nil
}

type safetyQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *PostgresService) CreateQualification(ctx context.Context, input CreateQualificationInput) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return MerchantQualification{}, err
	}
	if err := s.ready(); err != nil {
		return MerchantQualification{}, err
	}

	now := s.currentTime()
	qualification := MerchantQualification{
		ID:                       newSafetyUUID(),
		MerchantID:               strings.TrimSpace(input.MerchantID),
		LegalEntityName:          strings.TrimSpace(input.LegalEntityName),
		StoreName:                strings.TrimSpace(input.StoreName),
		BusinessLicenseNumber:    strings.TrimSpace(input.BusinessLicenseNumber),
		FoodPermitNumber:         strings.TrimSpace(input.FoodPermitNumber),
		RegisteredAddress:        strings.TrimSpace(input.RegisteredAddress),
		OperatingAddress:         strings.TrimSpace(input.OperatingAddress),
		BusinessLicenseIssuedAt:  input.BusinessLicenseIssuedAt,
		BusinessLicenseExpiresAt: input.BusinessLicenseExpiresAt,
		FoodPermitIssuedAt:       input.FoodPermitIssuedAt,
		FoodPermitExpiresAt:      input.FoodPermitExpiresAt,
		BusinessScope:            uniqueStrings(input.BusinessScope),
		Status:                   QualificationDraft,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if !isSafetyUUID(qualification.MerchantID) {
		return MerchantQualification{}, validationError(FieldError{"merchant_id", "must be a UUID"})
	}
	if err := ensureSafetyUUID(input.CreatedBy, "created_by"); err != nil {
		return MerchantQualification{}, err
	}
	qualification.Documents = make([]QualificationDocument, 0, len(input.Documents))
	for _, document := range input.Documents {
		document.ID = strings.TrimSpace(document.ID)
		if document.ID == "" {
			document.ID = newSafetyUUID()
		} else if !isSafetyUUID(document.ID) {
			return MerchantQualification{}, validationError(FieldError{"documents.id", "must be a UUID"})
		}
		document.Kind = QualificationDocumentKind(strings.TrimSpace(string(document.Kind)))
		document.URI = strings.TrimSpace(document.URI)
		document.SHA256 = strings.ToLower(strings.TrimSpace(document.SHA256))
		document.UploadedBy = strings.TrimSpace(document.UploadedBy)
		if document.UploadedBy == "" {
			document.UploadedBy = strings.TrimSpace(input.CreatedBy)
		}
		if err := ensureSafetyUUID(document.UploadedBy, "documents.uploaded_by"); err != nil {
			return MerchantQualification{}, err
		}
		if document.IssuedAt.IsZero() {
			switch document.Kind {
			case DocumentBusinessLicense:
				document.IssuedAt = qualification.BusinessLicenseIssuedAt
			case DocumentFoodPermit:
				document.IssuedAt = qualification.FoodPermitIssuedAt
			default:
				document.IssuedAt = now
			}
		}
		if document.UploadedAt.IsZero() {
			document.UploadedAt = now
		}
		qualification.Documents = append(qualification.Documents, document)
	}
	if err := ValidateQualification(qualification, now); err != nil {
		return MerchantQualification{}, err
	}

	businessScope, err := marshalSafetyJSON(qualification.BusinessScope, []string{})
	if err != nil {
		return MerchantQualification{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO merchant_qualifications (
			id, merchant_id, legal_entity_name, store_name,
			business_license_number, food_permit_number,
			registered_address, operating_address,
			business_license_issued_at, business_license_expires_at,
			food_permit_issued_at, food_permit_expires_at,
			business_scope, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14, $15, $15)
	`, qualification.ID, qualification.MerchantID, qualification.LegalEntityName,
		qualification.StoreName, qualification.BusinessLicenseNumber,
		qualification.FoodPermitNumber, qualification.RegisteredAddress,
		qualification.OperatingAddress, qualification.BusinessLicenseIssuedAt,
		qualification.BusinessLicenseExpiresAt, qualification.FoodPermitIssuedAt,
		qualification.FoodPermitExpiresAt, businessScope, qualification.Status, now)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	for _, document := range qualification.Documents {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO qualification_documents (
				id, qualification_id, kind, object_uri, sha256,
				issued_at, expires_at, uploaded_by, uploaded_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, document.ID, qualification.ID, document.Kind, document.URI,
			document.SHA256, document.IssuedAt, nullableTime(document.ExpiresAt),
			document.UploadedBy, document.UploadedAt)
		if err != nil {
			return MerchantQualification{}, normalizeSafetyDBError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	return qualification, nil
}

func (s *PostgresService) GetQualification(ctx context.Context, id string) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return MerchantQualification{}, err
	}
	if err := s.ready(); err != nil {
		return MerchantQualification{}, err
	}
	return s.loadQualification(ctx, s.db, id, false)
}

func (s *PostgresService) ListQualifications(ctx context.Context, merchantID string, status QualificationStatus) ([]MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if err := s.ready(); err != nil {
		return nil, err
	}
	if merchantID != "" && !isSafetyUUID(merchantID) {
		return nil, validationError(FieldError{"merchant_id", "must be a UUID"})
	}
	query := `
		SELECT id::text
		FROM merchant_qualifications
	`
	args := make([]any, 0, 2)
	conditions := make([]string, 0, 2)
	if merchantID != "" {
		args = append(args, merchantID)
		conditions = append(conditions, fmt.Sprintf("merchant_id = $%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeSafetyDBError(err)
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, normalizeSafetyDBError(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, normalizeSafetyDBError(err)
	}
	rows.Close()
	result := make([]MerchantQualification, 0, len(ids))
	for _, id := range ids {
		qualification, err := s.loadQualification(ctx, s.db, id, false)
		if err != nil {
			return nil, err
		}
		result = append(result, qualification)
	}
	return result, nil
}

func (s *PostgresService) SubmitQualification(ctx context.Context, id string, input SubmitQualificationInput) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return MerchantQualification{}, err
	}
	if err := s.ready(); err != nil {
		return MerchantQualification{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return MerchantQualification{}, validationError(FieldError{"actor_id", "is required"})
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	qualification, err := s.loadQualification(ctx, tx, id, true)
	if err != nil {
		return MerchantQualification{}, err
	}
	if !CanTransitionQualification(qualification.Status, QualificationPendingReview) {
		return MerchantQualification{}, invalidTransition(string(qualification.Status), string(QualificationPendingReview))
	}
	if err := ValidateQualificationForSubmission(qualification, s.currentTime()); err != nil {
		return MerchantQualification{}, err
	}
	if err := s.validateEvidenceRefsTx(ctx, tx, EvidenceSubject{Type: EvidenceQualification, ID: id}, input.EvidenceIDs); err != nil {
		return MerchantQualification{}, err
	}
	now := s.currentTime()
	_, err = tx.ExecContext(ctx, `
		UPDATE merchant_qualifications
		SET status = $2, submitted_at = $3, updated_at = $3
		WHERE id = $1
	`, id, QualificationPendingReview, now)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	qualification, err = s.loadQualification(ctx, tx, id, false)
	if err != nil {
		return MerchantQualification{}, err
	}
	if err := tx.Commit(); err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	return qualification, nil
}

func (s *PostgresService) ReviewQualification(ctx context.Context, id string, input ReviewQualificationInput) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return MerchantQualification{}, err
	}
	if err := s.ready(); err != nil {
		return MerchantQualification{}, err
	}
	if strings.TrimSpace(input.ReviewerID) == "" {
		return MerchantQualification{}, validationError(FieldError{"reviewer_id", "is required"})
	}
	if input.Status != QualificationApproved && input.Status != QualificationRejected {
		return MerchantQualification{}, validationError(FieldError{"status", "must be APPROVED or REJECTED"})
	}
	if input.Status == QualificationRejected && strings.TrimSpace(input.Reason) == "" {
		return MerchantQualification{}, validationError(FieldError{"reason", "is required when rejecting"})
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	qualification, err := s.loadQualification(ctx, tx, id, true)
	if err != nil {
		return MerchantQualification{}, err
	}
	if qualification.Status != QualificationPendingReview {
		return MerchantQualification{}, invalidTransition(string(qualification.Status), string(input.Status))
	}
	if err := s.validateEvidenceRefsTx(ctx, tx, EvidenceSubject{Type: EvidenceQualification, ID: id}, input.EvidenceIDs); err != nil {
		return MerchantQualification{}, err
	}
	now := s.currentTime()
	if input.Status == QualificationApproved {
		if qualification.SiteInspection == nil || qualification.SiteInspection.Result != SiteInspectionPassed {
			return MerchantQualification{}, fmt.Errorf("%w: a passed site inspection is required for approval", ErrInvalidState)
		}
		if !qualification.BusinessLicenseExpiresAt.After(now) || !qualification.FoodPermitExpiresAt.After(now) {
			return MerchantQualification{}, fmt.Errorf("%w: permits must be valid for approval", ErrInvalidState)
		}
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE merchant_qualifications
		SET status = $2, reviewed_at = $3, updated_at = $3
		WHERE id = $1
	`, id, input.Status, now)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	qualification, err = s.loadQualification(ctx, tx, id, false)
	if err != nil {
		return MerchantQualification{}, err
	}
	if err := tx.Commit(); err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	return qualification, nil
}

func (s *PostgresService) RecordSiteInspection(ctx context.Context, id string, input SiteInspectionInput) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return MerchantQualification{}, err
	}
	if err := s.ready(); err != nil {
		return MerchantQualification{}, err
	}
	if err := ensureSafetyUUID(input.InspectorID, "inspector_id"); err != nil {
		return MerchantQualification{}, err
	}
	if !input.Result.Valid() {
		return MerchantQualification{}, validationError(FieldError{"result", "is invalid"})
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	qualification, err := s.loadQualification(ctx, tx, id, true)
	if err != nil {
		return MerchantQualification{}, err
	}
	if qualification.Status != QualificationDraft && qualification.Status != QualificationPendingReview {
		return MerchantQualification{}, fmt.Errorf("%w: inspection is not allowed in %s", ErrInvalidState, qualification.Status)
	}
	if err := s.validateEvidenceRefsTx(ctx, tx, EvidenceSubject{Type: EvidenceQualification, ID: id}, input.EvidenceIDs); err != nil {
		return MerchantQualification{}, err
	}
	now := s.currentTime()
	inspectedAt := input.InspectedAt
	if inspectedAt.IsZero() {
		inspectedAt = now
	}
	evidenceIDs, err := marshalSafetyJSON(uniqueStrings(input.EvidenceIDs), []string{})
	if err != nil {
		return MerchantQualification{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO site_inspections (
			id, qualification_id, inspector_id, result, notes, evidence_ids, inspected_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
	`, newSafetyUUID(), id, input.InspectorID, input.Result,
		nullableString(input.Notes), evidenceIDs, inspectedAt)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE merchant_qualifications SET updated_at = $2 WHERE id = $1
	`, id, now)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	qualification, err = s.loadQualification(ctx, tx, id, false)
	if err != nil {
		return MerchantQualification{}, err
	}
	if err := tx.Commit(); err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	return qualification, nil
}

func (s *PostgresService) CreateBatch(ctx context.Context, input CreateBatchInput) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	if err := s.ready(); err != nil {
		return FoodBatch{}, err
	}
	if err := ensureSafetyUUID(input.MerchantID, "merchant_id"); err != nil {
		return FoodBatch{}, err
	}
	if err := ensureSafetyUUID(input.CreatedBy, "created_by"); err != nil {
		return FoodBatch{}, err
	}
	if err := ensureSafetyUUID(input.ProductID, "product_id"); err != nil {
		return FoodBatch{}, err
	}
	if input.CampaignID != "" {
		if err := ensureSafetyUUID(input.CampaignID, "campaign_id"); err != nil {
			return FoodBatch{}, err
		}
	}
	now := s.currentTime()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	var qualificationID string
	err = tx.QueryRowContext(ctx, `
		SELECT id::text
		FROM merchant_qualifications
		WHERE merchant_id = $1
		  AND status = $2
		  AND business_license_expires_at > $3
		  AND food_permit_expires_at > $3
		ORDER BY updated_at DESC
		LIMIT 1
		FOR UPDATE
	`, input.MerchantID, QualificationApproved, now).Scan(&qualificationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			return FoodBatch{}, fmt.Errorf("%w: merchant qualification is not approved and current", ErrInvalidState)
		}
		_ = tx.Rollback()
		return FoodBatch{}, normalizeSafetyDBError(err)
	}

	batch := FoodBatch{
		ID:               newSafetyUUID(),
		MerchantID:       strings.TrimSpace(input.MerchantID),
		ProductID:        strings.TrimSpace(input.ProductID),
		CampaignID:       strings.TrimSpace(input.CampaignID),
		ProductionDate:   input.ProductionDate,
		ShelfLifeMinutes: input.ShelfLifeMinutes,
		StorageCondition: strings.TrimSpace(input.StorageCondition),
		QuantityPlanned:  input.QuantityPlanned,
		UnitWeightGrams:  input.UnitWeightGrams,
		Specification:    input.Specification,
		IngredientLots:   input.IngredientLots,
		Status:           BatchDraft,
		CreatedBy:        strings.TrimSpace(input.CreatedBy),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := ValidateBatch(batch, now); err != nil {
		return FoodBatch{}, err
	}
	specification, err := marshalSafetyJSON(batch.Specification, ProductSpec{})
	if err != nil {
		return FoodBatch{}, err
	}
	ingredientLots, err := marshalSafetyJSON(batch.IngredientLots, []IngredientLot{})
	if err != nil {
		return FoodBatch{}, err
	}
	var campaignID any
	if batch.CampaignID != "" {
		campaignID = batch.CampaignID
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO food_batches (
			id, merchant_id, campaign_id, product_id, production_date,
			shelf_life_minutes, storage_condition, quantity_planned,
			quantity_produced, quantity_remaining, unit_weight_grams,
			specification, ingredient_lots, status, created_by, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, 0, $9, $10::jsonb, $11::jsonb, $12, $13, $14, $14)
	`, batch.ID, batch.MerchantID, campaignID, batch.ProductID,
		batch.ProductionDate, batch.ShelfLifeMinutes, batch.StorageCondition,
		batch.QuantityPlanned, batch.UnitWeightGrams, specification,
		ingredientLots, batch.Status, batch.CreatedBy, now)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	batch, err = s.loadBatch(ctx, tx, batch.ID, false)
	if err != nil {
		return FoodBatch{}, err
	}
	if err := tx.Commit(); err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	return batch, nil
}

func (s *PostgresService) GetBatch(ctx context.Context, id string) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	if err := s.ready(); err != nil {
		return FoodBatch{}, err
	}
	return s.loadBatch(ctx, s.db, id, false)
}

func (s *PostgresService) ListBatches(ctx context.Context, merchantID string, status FoodBatchStatus) ([]FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if err := s.ready(); err != nil {
		return nil, err
	}
	if merchantID != "" && !isSafetyUUID(merchantID) {
		return nil, validationError(FieldError{"merchant_id", "must be a UUID"})
	}
	query := "SELECT id::text FROM food_batches"
	args := make([]any, 0, 2)
	conditions := make([]string, 0, 2)
	if merchantID != "" {
		args = append(args, merchantID)
		conditions = append(conditions, fmt.Sprintf("merchant_id = $%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY production_date DESC, created_at DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeSafetyDBError(err)
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, normalizeSafetyDBError(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, normalizeSafetyDBError(err)
	}
	rows.Close()
	result := make([]FoodBatch, 0, len(ids))
	for _, id := range ids {
		batch, err := s.loadBatch(ctx, s.db, id, false)
		if err != nil {
			return nil, err
		}
		result = append(result, batch)
	}
	return result, nil
}

func (s *PostgresService) RecordBatchProduction(ctx context.Context, id string, input RecordProductionInput) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	if err := s.ready(); err != nil {
		return FoodBatch{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return FoodBatch{}, validationError(FieldError{"actor_id", "is required"})
	}
	if input.Quantity <= 0 {
		return FoodBatch{}, validationError(FieldError{"quantity", "must be positive"})
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	batch, err := s.loadBatch(ctx, tx, id, true)
	if err != nil {
		return FoodBatch{}, err
	}
	if batch.Status != BatchProducing {
		return FoodBatch{}, fmt.Errorf("%w: production can only be recorded while PRODUCING", ErrInvalidState)
	}
	if input.Quantity > batch.QuantityPlanned {
		return FoodBatch{}, validationError(FieldError{"quantity", "cannot exceed quantity planned"})
	}
	producedAt := input.ProducedAt
	if producedAt.IsZero() {
		producedAt = s.currentTime()
	}
	expiresAt := input.ExpiresAt
	shelfLife := batch.ShelfLifeMinutes
	if input.ShelfLifeMinutes > 0 {
		shelfLife = input.ShelfLifeMinutes
	}
	if expiresAt.IsZero() {
		expiresAt = producedAt.Add(time.Duration(shelfLife) * time.Minute)
	}
	if !expiresAt.After(producedAt) {
		return FoodBatch{}, validationError(FieldError{"expires_at", "must be after produced_at"})
	}
	now := s.currentTime()
	_, err = tx.ExecContext(ctx, `
		UPDATE food_batches
		SET produced_at = $2, expires_at = $3, shelf_life_minutes = $4,
		    quantity_produced = $5, quantity_remaining = $5, updated_at = $6
		WHERE id = $1
	`, id, producedAt, expiresAt, shelfLife, input.Quantity, now)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	batch, err = s.loadBatch(ctx, tx, id, false)
	if err != nil {
		return FoodBatch{}, err
	}
	if err := tx.Commit(); err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	return batch, nil
}

func (s *PostgresService) AssociateOrders(ctx context.Context, id string, input AssociateOrdersInput) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	if err := s.ready(); err != nil {
		return FoodBatch{}, err
	}
	if err := ensureSafetyUUID(input.ActorID, "actor_id"); err != nil {
		return FoodBatch{}, err
	}
	if len(input.Orders) == 0 {
		return FoodBatch{}, validationError(FieldError{"orders", "at least one order is required"})
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	batch, err := s.loadBatch(ctx, tx, id, true)
	if err != nil {
		return FoodBatch{}, err
	}
	switch batch.Status {
	case BatchDraft, BatchScheduled, BatchProducing, BatchPacked, BatchReadyForHandoff:
	default:
		return FoodBatch{}, fmt.Errorf("%w: orders cannot be associated with %s batch", ErrInvalidState, batch.Status)
	}
	totalQuantity := 0
	for _, order := range batch.Orders {
		totalQuantity += order.Quantity
	}
	orders := append([]OrderAssociationInput(nil), input.Orders...)
	for _, order := range orders {
		if err := ensureSafetyUUID(order.OrderID, "orders.order_id"); err != nil {
			return FoodBatch{}, err
		}
	}
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].OrderID < orders[j].OrderID
	})
	added := make(map[string]struct{}, len(orders))
	now := s.currentTime()
	for _, order := range orders {
		if order.Quantity <= 0 {
			return FoodBatch{}, validationError(FieldError{"orders.quantity", "must be positive"})
		}
		if _, exists := added[order.OrderID]; exists {
			return FoodBatch{}, fmt.Errorf("%w: order %s is duplicated in request", ErrConflict, order.OrderID)
		}
		added[order.OrderID] = struct{}{}
		if _, err := tx.ExecContext(ctx, `
			SELECT pg_advisory_xact_lock(hashtextextended($1, 0))
		`, order.OrderID); err != nil {
			return FoodBatch{}, normalizeSafetyDBError(err)
		}
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM orders WHERE id = $1)`, order.OrderID).Scan(&exists); err != nil {
			return FoodBatch{}, normalizeSafetyDBError(err)
		}
		if !exists {
			return FoodBatch{}, ErrNotFound
		}
		var assignedBatch string
		err := tx.QueryRowContext(ctx, `
			SELECT batch_id::text FROM batch_order_associations
			WHERE order_id = $1
			ORDER BY linked_at
			LIMIT 1
			FOR UPDATE
		`, order.OrderID).Scan(&assignedBatch)
		if err == nil {
			return FoodBatch{}, fmt.Errorf("%w: order %s already belongs to batch %s", ErrConflict, order.OrderID, assignedBatch)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return FoodBatch{}, normalizeSafetyDBError(err)
		}
		if totalQuantity+order.Quantity > batch.QuantityPlanned {
			return FoodBatch{}, fmt.Errorf("%w: associated order quantity exceeds planned batch quantity", ErrConflict)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO batch_order_associations (batch_id, order_id, quantity, linked_by, linked_at)
			VALUES ($1, $2, $3, $4, $5)
		`, id, order.OrderID, order.Quantity, input.ActorID, now)
		if err != nil {
			return FoodBatch{}, normalizeSafetyDBError(err)
		}
		totalQuantity += order.Quantity
	}
	_, err = tx.ExecContext(ctx, `UPDATE food_batches SET updated_at = $2 WHERE id = $1`, id, now)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	batch, err = s.loadBatch(ctx, tx, id, false)
	if err != nil {
		return FoodBatch{}, err
	}
	if err := tx.Commit(); err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	return batch, nil
}

func (s *PostgresService) TransitionBatch(ctx context.Context, id string, target FoodBatchStatus, input TransitionInput) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	if err := s.ready(); err != nil {
		return FoodBatch{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return FoodBatch{}, validationError(FieldError{"actor_id", "is required"})
	}
	if !target.Valid() {
		return FoodBatch{}, validationError(FieldError{"status", "is invalid"})
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	batch, err := s.loadBatch(ctx, tx, id, true)
	if err != nil {
		return FoodBatch{}, err
	}
	if !CanTransitionBatch(batch.Status, target) {
		return FoodBatch{}, invalidTransition(string(batch.Status), string(target))
	}
	if target == BatchPacked && (batch.ProducedAt == nil || batch.ExpiresAt == nil || batch.QuantityProduced <= 0) {
		return FoodBatch{}, fmt.Errorf("%w: production data is required before packing", ErrInvalidState)
	}
	if target == BatchReadyForHandoff && batch.QuantityProduced <= 0 {
		return FoodBatch{}, fmt.Errorf("%w: a batch must have produced quantity before handoff", ErrInvalidState)
	}
	if target == BatchInDelivery && len(batch.Orders) == 0 {
		return FoodBatch{}, fmt.Errorf("%w: orders must be associated before delivery", ErrInvalidState)
	}
	if err := s.validateEvidenceRefsTx(ctx, tx, EvidenceSubject{Type: EvidenceBatch, ID: id}, input.EvidenceIDs); err != nil {
		return FoodBatch{}, err
	}
	now := s.currentTime()
	_, err = tx.ExecContext(ctx, `
		UPDATE food_batches SET status = $2, updated_at = $3 WHERE id = $1
	`, id, target, now)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	batch, err = s.loadBatch(ctx, tx, id, false)
	if err != nil {
		return FoodBatch{}, err
	}
	if err := tx.Commit(); err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	return batch, nil
}

func (s *PostgresService) CreateIncident(ctx context.Context, input CreateIncidentInput) (FoodSafetyIncident, error) {
	if err := checkContext(ctx); err != nil {
		return FoodSafetyIncident{}, err
	}
	if err := s.ready(); err != nil {
		return FoodSafetyIncident{}, err
	}
	if len(input.EvidenceIDs) > 0 {
		return FoodSafetyIncident{}, fmt.Errorf("%w: add incident evidence after creation", ErrConflict)
	}
	if err := ensureSafetyUUID(input.MerchantID, "merchant_id"); err != nil {
		return FoodSafetyIncident{}, err
	}
	if err := ensureSafetyUUID(input.ReportedBy, "reported_by"); err != nil {
		return FoodSafetyIncident{}, err
	}
	batchIDs := uniqueStrings(input.BatchIDs)
	for _, batchID := range batchIDs {
		if !isSafetyUUID(batchID) {
			return FoodSafetyIncident{}, validationError(FieldError{"batch_ids", "must contain UUIDs"})
		}
	}
	reportedAt := input.ReportedAt
	if reportedAt.IsZero() {
		reportedAt = s.currentTime()
	}
	incident := FoodSafetyIncident{
		ID:               newSafetyUUID(),
		MerchantID:       strings.TrimSpace(input.MerchantID),
		ReportedBy:       strings.TrimSpace(input.ReportedBy),
		Category:         strings.TrimSpace(input.Category),
		Severity:         input.Severity,
		Title:            strings.TrimSpace(input.Title),
		Description:      strings.TrimSpace(input.Description),
		BatchIDs:         batchIDs,
		OrderIDs:         uniqueStrings(input.OrderIDs),
		Status:           IncidentReported,
		RegulatoryReport: cloneRegulatoryReport(input.RegulatoryReport),
		ReportedAt:       reportedAt,
		CreatedAt:        s.currentTime(),
		UpdatedAt:        s.currentTime(),
	}
	if err := ValidateIncident(incident); err != nil {
		return FoodSafetyIncident{}, err
	}
	batchJSON, err := marshalSafetyJSON(incident.BatchIDs, []string{})
	if err != nil {
		return FoodSafetyIncident{}, err
	}
	orderJSON, err := marshalSafetyJSON(incident.OrderIDs, []string{})
	if err != nil {
		return FoodSafetyIncident{}, err
	}
	regulatoryJSON, err := marshalSafetyJSON(incident.RegulatoryReport, RegulatoryReport{})
	if err != nil {
		return FoodSafetyIncident{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FoodSafetyIncident{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, batchID := range incident.BatchIDs {
		var merchantID string
		err := tx.QueryRowContext(ctx, `
			SELECT merchant_id::text FROM food_batches WHERE id = $1
		`, batchID).Scan(&merchantID)
		if errors.Is(err, sql.ErrNoRows) {
			return FoodSafetyIncident{}, ErrNotFound
		}
		if err != nil {
			return FoodSafetyIncident{}, normalizeSafetyDBError(err)
		}
		if merchantID != incident.MerchantID {
			return FoodSafetyIncident{}, fmt.Errorf("%w: batch %s belongs to another merchant", ErrConflict, batchID)
		}
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO safety_incidents (
			id, merchant_id, reported_by, category, severity, title, description,
			batch_ids, order_ids, status, regulatory_report, reported_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10, $11::jsonb, $12, $13, $13)
	`, incident.ID, incident.MerchantID, incident.ReportedBy, incident.Category,
		incident.Severity, incident.Title, incident.Description, batchJSON,
		orderJSON, incident.Status, regulatoryJSON, incident.ReportedAt,
		incident.CreatedAt)
	if err != nil {
		return FoodSafetyIncident{}, normalizeSafetyDBError(err)
	}
	incident, err = s.loadIncident(ctx, tx, incident.ID, false)
	if err != nil {
		return FoodSafetyIncident{}, err
	}
	if err := tx.Commit(); err != nil {
		return FoodSafetyIncident{}, normalizeSafetyDBError(err)
	}
	return incident, nil
}

func (s *PostgresService) GetIncident(ctx context.Context, id string) (FoodSafetyIncident, error) {
	if err := checkContext(ctx); err != nil {
		return FoodSafetyIncident{}, err
	}
	if err := s.ready(); err != nil {
		return FoodSafetyIncident{}, err
	}
	return s.loadIncident(ctx, s.db, id, false)
}

func (s *PostgresService) ListIncidents(ctx context.Context, merchantID string, status IncidentStatus) ([]FoodSafetyIncident, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if err := s.ready(); err != nil {
		return nil, err
	}
	if merchantID != "" && !isSafetyUUID(merchantID) {
		return nil, validationError(FieldError{"merchant_id", "must be a UUID"})
	}
	query := "SELECT id::text FROM safety_incidents"
	args := make([]any, 0, 2)
	conditions := make([]string, 0, 2)
	if merchantID != "" {
		args = append(args, merchantID)
		conditions = append(conditions, fmt.Sprintf("merchant_id = $%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY reported_at DESC, created_at DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeSafetyDBError(err)
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, normalizeSafetyDBError(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, normalizeSafetyDBError(err)
	}
	rows.Close()
	result := make([]FoodSafetyIncident, 0, len(ids))
	for _, id := range ids {
		incident, err := s.loadIncident(ctx, s.db, id, false)
		if err != nil {
			return nil, err
		}
		result = append(result, incident)
	}
	return result, nil
}

func (s *PostgresService) TransitionIncident(ctx context.Context, id string, target IncidentStatus, input IncidentTransitionInput) (FoodSafetyIncident, error) {
	if err := checkContext(ctx); err != nil {
		return FoodSafetyIncident{}, err
	}
	if err := s.ready(); err != nil {
		return FoodSafetyIncident{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return FoodSafetyIncident{}, validationError(FieldError{"actor_id", "is required"})
	}
	if !target.Valid() {
		return FoodSafetyIncident{}, validationError(FieldError{"status", "is invalid"})
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FoodSafetyIncident{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	incident, err := s.loadIncident(ctx, tx, id, true)
	if err != nil {
		return FoodSafetyIncident{}, err
	}
	if !CanTransitionIncident(incident.Status, target) {
		return FoodSafetyIncident{}, invalidTransition(string(incident.Status), string(target))
	}
	if target == IncidentContained && strings.TrimSpace(input.ContainmentAction) == "" {
		return FoodSafetyIncident{}, validationError(FieldError{"containment_action", "is required when containing an incident"})
	}
	if target == IncidentResolved && strings.TrimSpace(input.ResolutionSummary) == "" {
		return FoodSafetyIncident{}, validationError(FieldError{"resolution_summary", "is required when resolving an incident"})
	}
	if input.RegulatoryReport != nil && input.RegulatoryReport.Reported && input.RegulatoryReport.ReportedAt == nil {
		return FoodSafetyIncident{}, validationError(FieldError{"regulatory_report.reported_at", "is required when reported is true"})
	}
	if err := s.validateEvidenceRefsTx(ctx, tx, EvidenceSubject{Type: EvidenceIncident, ID: id}, input.EvidenceIDs); err != nil {
		return FoodSafetyIncident{}, err
	}
	if target == IncidentContained {
		if err := s.quarantineIncidentBatchesTx(ctx, tx, incident); err != nil {
			return FoodSafetyIncident{}, err
		}
	}
	now := s.currentTime()
	containmentAction := incident.ContainmentAction
	investigationSummary := incident.InvestigationSummary
	resolutionSummary := incident.ResolutionSummary
	containedAt := nullableTime(incident.ContainedAt)
	resolvedAt := nullableTime(incident.ResolvedAt)
	closedAt := nullableTime(incident.ClosedAt)
	regulatoryReport := incident.RegulatoryReport
	if input.RegulatoryReport != nil {
		regulatoryReport = cloneRegulatoryReport(*input.RegulatoryReport)
	}
	if target == IncidentContained {
		containmentAction = strings.TrimSpace(input.ContainmentAction)
		containedAt = now
	}
	if target == IncidentInvestigating {
		investigationSummary = strings.TrimSpace(input.InvestigationSummary)
	}
	if target == IncidentResolved {
		resolutionSummary = strings.TrimSpace(input.ResolutionSummary)
		resolvedAt = now
	}
	if target == IncidentClosed {
		closedAt = now
	}
	regulatoryJSON, err := marshalSafetyJSON(regulatoryReport, RegulatoryReport{})
	if err != nil {
		return FoodSafetyIncident{}, err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE safety_incidents
		SET status = $2, containment_action = $3, investigation_summary = $4,
		    resolution_summary = $5, regulatory_report = $6::jsonb,
		    contained_at = $7, resolved_at = $8, closed_at = $9, updated_at = $10
		WHERE id = $1
	`, id, target, nullableString(containmentAction),
		nullableString(investigationSummary), nullableString(resolutionSummary),
		regulatoryJSON, containedAt, resolvedAt, closedAt, now)
	if err != nil {
		return FoodSafetyIncident{}, normalizeSafetyDBError(err)
	}
	incident, err = s.loadIncident(ctx, tx, id, false)
	if err != nil {
		return FoodSafetyIncident{}, err
	}
	if err := tx.Commit(); err != nil {
		return FoodSafetyIncident{}, normalizeSafetyDBError(err)
	}
	return incident, nil
}

func (s *PostgresService) CreateRecall(context.Context, CreateRecallInput) (Recall, error) {
	return Recall{}, fmt.Errorf("%w: recall persistence requires the next schema revision", ErrInvalidState)
}

func (s *PostgresService) GetRecall(context.Context, string) (Recall, error) {
	return Recall{}, fmt.Errorf("%w: recall persistence requires the next schema revision", ErrInvalidState)
}

func (s *PostgresService) ListRecalls(context.Context, string, RecallStatus) ([]Recall, error) {
	return nil, fmt.Errorf("%w: recall persistence requires the next schema revision", ErrInvalidState)
}

func (s *PostgresService) TransitionRecall(context.Context, string, RecallStatus, RecallTransitionInput) (Recall, error) {
	return Recall{}, fmt.Errorf("%w: recall persistence requires the next schema revision", ErrInvalidState)
}

func (s *PostgresService) CreateDisposition(context.Context, CreateDispositionInput) (Disposition, error) {
	return Disposition{}, fmt.Errorf("%w: disposition persistence requires the next schema revision", ErrInvalidState)
}

func (s *PostgresService) GetDisposition(context.Context, string) (Disposition, error) {
	return Disposition{}, fmt.Errorf("%w: disposition persistence requires the next schema revision", ErrInvalidState)
}

func (s *PostgresService) TransitionDisposition(context.Context, string, DispositionStatus, TransitionInput) (Disposition, error) {
	return Disposition{}, fmt.Errorf("%w: disposition persistence requires the next schema revision", ErrInvalidState)
}

func (s *PostgresService) AddEvidence(ctx context.Context, subject EvidenceSubject, input AddEvidenceInput) (Evidence, error) {
	if err := checkContext(ctx); err != nil {
		return Evidence{}, err
	}
	if err := s.ready(); err != nil {
		return Evidence{}, err
	}
	subject.Type = EvidenceSubjectType(strings.TrimSpace(string(subject.Type)))
	subject.ID = strings.TrimSpace(subject.ID)
	if !subject.Type.Valid() {
		return Evidence{}, validationError(FieldError{"subject.type", "is invalid"})
	}
	if err := ensureSafetyUUID(subject.ID, "subject.id"); err != nil {
		return Evidence{}, err
	}
	if subject.Type == EvidenceRecall || subject.Type == EvidenceDisposition {
		return Evidence{}, fmt.Errorf("%w: evidence for recalls and dispositions requires the next schema revision", ErrInvalidState)
	}
	now := s.currentTime()
	capturedAt := input.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = now
	}
	evidence := Evidence{
		ID:          newSafetyUUID(),
		Subject:     subject,
		Kind:        input.Kind,
		URI:         strings.TrimSpace(input.URI),
		SHA256:      strings.ToLower(strings.TrimSpace(input.SHA256)),
		MimeType:    strings.TrimSpace(input.MimeType),
		CapturedBy:  strings.TrimSpace(input.CapturedBy),
		CapturedAt:  capturedAt,
		Description: strings.TrimSpace(input.Description),
		Metadata:    cloneStringMap(input.Metadata),
		CreatedAt:   now,
	}
	if err := validateEvidencePayload(evidence.URI, evidence.SHA256); err != nil {
		return Evidence{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Evidence{}, normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.lockEvidenceSubjectTx(ctx, tx, subject); err != nil {
		return Evidence{}, err
	}
	var sequence uint64
	var previousHash string
	err = tx.QueryRowContext(ctx, `
		SELECT sequence_no, current_hash
		FROM safety_evidences
		WHERE subject_type = $1 AND subject_id = $2
		ORDER BY sequence_no DESC
		LIMIT 1
	`, subject.Type, subject.ID).Scan(&sequence, &previousHash)
	if errors.Is(err, sql.ErrNoRows) {
		sequence = 1
		previousHash = ""
	} else if err != nil {
		return Evidence{}, normalizeSafetyDBError(err)
	} else {
		sequence++
	}
	evidence.Sequence = sequence
	evidence.PreviousHash = previousHash
	evidence.Hash = evidenceHash(evidence)
	if err := ValidateEvidence(evidence); err != nil {
		return Evidence{}, err
	}
	metadata, err := marshalPersistedEvidenceMetadata(evidence)
	if err != nil {
		return Evidence{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO safety_evidences (
			id, subject_type, subject_id, evidence_type, object_uri, sha256,
			sequence_no, previous_hash, current_hash, metadata,
			collected_by, collected_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12)
	`, evidence.ID, evidence.Subject.Type, evidence.Subject.ID, evidence.Kind,
		evidence.URI, evidence.SHA256, evidence.Sequence, nullableString(evidence.PreviousHash),
		evidence.Hash, metadata, evidence.CapturedBy, evidence.CapturedAt)
	if err != nil {
		return Evidence{}, normalizeSafetyDBError(err)
	}
	if subject.Type == EvidenceIncident {
		ids, _ := marshalSafetyJSON([]string{evidence.ID}, []string{})
		_, err = tx.ExecContext(ctx, `
			UPDATE safety_incidents
			SET evidence_ids = evidence_ids || $2::jsonb, updated_at = $3
			WHERE id = $1
		`, subject.ID, ids, now)
		if err != nil {
			return Evidence{}, normalizeSafetyDBError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Evidence{}, normalizeSafetyDBError(err)
	}
	return evidence, nil
}

func (s *PostgresService) ListEvidence(ctx context.Context, subject EvidenceSubject) ([]Evidence, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	if err := s.ready(); err != nil {
		return nil, err
	}
	subject.Type = EvidenceSubjectType(strings.TrimSpace(string(subject.Type)))
	subject.ID = strings.TrimSpace(subject.ID)
	if !subject.Type.Valid() {
		return nil, validationError(FieldError{"subject.type", "is invalid"})
	}
	if err := ensureSafetyUUID(subject.ID, "subject.id"); err != nil {
		return nil, err
	}
	if subject.Type == EvidenceRecall || subject.Type == EvidenceDisposition {
		return nil, fmt.Errorf("%w: evidence for recalls and dispositions requires the next schema revision", ErrInvalidState)
	}
	if err := s.ensureEvidenceSubjectExists(ctx, s.db, subject); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id::text, sequence_no, previous_hash, current_hash,
		       evidence_type, object_uri, sha256, metadata,
		       collected_by::text, collected_at
		FROM safety_evidences
		WHERE subject_type = $1 AND subject_id = $2
		ORDER BY sequence_no
	`, subject.Type, subject.ID)
	if err != nil {
		return nil, normalizeSafetyDBError(err)
	}
	defer rows.Close()
	result := make([]Evidence, 0)
	for rows.Next() {
		evidence, err := scanPersistedEvidence(rows, subject)
		if err != nil {
			return nil, err
		}
		result = append(result, evidence)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeSafetyDBError(err)
	}
	return result, nil
}

func (s *PostgresService) VerifyEvidenceChain(ctx context.Context, subject EvidenceSubject) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := s.ready(); err != nil {
		return err
	}
	subject.Type = EvidenceSubjectType(strings.TrimSpace(string(subject.Type)))
	subject.ID = strings.TrimSpace(subject.ID)
	if !subject.Type.Valid() {
		return validationError(FieldError{"subject.type", "is invalid"})
	}
	if err := ensureSafetyUUID(subject.ID, "subject.id"); err != nil {
		return err
	}
	if subject.Type == EvidenceRecall || subject.Type == EvidenceDisposition {
		return fmt.Errorf("%w: evidence for recalls and dispositions requires the next schema revision", ErrInvalidState)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return normalizeSafetyDBError(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.lockEvidenceSubjectTx(ctx, tx, subject); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id::text, sequence_no, previous_hash, current_hash,
		       evidence_type, object_uri, sha256, metadata,
		       collected_by::text, collected_at
		FROM safety_evidences
		WHERE subject_type = $1 AND subject_id = $2
		ORDER BY sequence_no
	`, subject.Type, subject.ID)
	if err != nil {
		return normalizeSafetyDBError(err)
	}
	evidence := make([]Evidence, 0)
	for rows.Next() {
		item, err := scanPersistedEvidence(rows, subject)
		if err != nil {
			rows.Close()
			return err
		}
		evidence = append(evidence, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return normalizeSafetyDBError(err)
	}
	rows.Close()
	var previous string
	for index, item := range evidence {
		if item.Sequence != uint64(index+1) || item.PreviousHash != previous ||
			item.Hash != evidenceHash(item) {
			return fmt.Errorf("%w: evidence chain is invalid at sequence %d", ErrConflict, index+1)
		}
		previous = item.Hash
	}
	if err := tx.Commit(); err != nil {
		return normalizeSafetyDBError(err)
	}
	return nil
}

func (s *PostgresService) loadQualification(ctx context.Context, queryer safetyQueryer, id string, lock bool) (MerchantQualification, error) {
	if !isSafetyUUID(id) {
		return MerchantQualification{}, validationError(FieldError{"id", "must be a UUID"})
	}
	query := `
		SELECT id::text, merchant_id::text, legal_entity_name, store_name,
		       business_license_number, food_permit_number,
		       registered_address, operating_address,
		       business_license_issued_at, business_license_expires_at,
		       food_permit_issued_at, food_permit_expires_at,
		       business_scope, status, submitted_at, reviewed_at,
		       created_at, updated_at
		FROM merchant_qualifications
		WHERE id = $1
	`
	if lock {
		query += " FOR UPDATE"
	}
	var qualification MerchantQualification
	var businessScope []byte
	var submittedAt, reviewedAt sql.NullTime
	err := queryer.QueryRowContext(ctx, query, id).Scan(
		&qualification.ID, &qualification.MerchantID, &qualification.LegalEntityName,
		&qualification.StoreName, &qualification.BusinessLicenseNumber,
		&qualification.FoodPermitNumber, &qualification.RegisteredAddress,
		&qualification.OperatingAddress, &qualification.BusinessLicenseIssuedAt,
		&qualification.BusinessLicenseExpiresAt, &qualification.FoodPermitIssuedAt,
		&qualification.FoodPermitExpiresAt, &businessScope, &qualification.Status,
		&submittedAt, &reviewedAt, &qualification.CreatedAt, &qualification.UpdatedAt,
	)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	if err := unmarshalSafetyJSON(businessScope, &qualification.BusinessScope); err != nil {
		return MerchantQualification{}, err
	}
	qualification.SubmittedAt = nullTimePointer(submittedAt)
	qualification.ReviewedAt = nullTimePointer(reviewedAt)

	rows, err := queryer.QueryContext(ctx, `
		SELECT id::text, kind, object_uri, sha256, issued_at, expires_at,
		       uploaded_by::text, uploaded_at
		FROM qualification_documents
		WHERE qualification_id = $1
		ORDER BY uploaded_at, id
	`, id)
	if err != nil {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	for rows.Next() {
		var document QualificationDocument
		var expiresAt sql.NullTime
		if err := rows.Scan(&document.ID, &document.Kind, &document.URI,
			&document.SHA256, &document.IssuedAt, &expiresAt,
			&document.UploadedBy, &document.UploadedAt); err != nil {
			rows.Close()
			return MerchantQualification{}, normalizeSafetyDBError(err)
		}
		document.ExpiresAt = nullTimePointer(expiresAt)
		qualification.Documents = append(qualification.Documents, document)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	rows.Close()

	var inspection SiteInspection
	var inspectionEvidence []byte
	var inspectionNotes sql.NullString
	var inspectionFound bool
	var inspectionResult string
	err = queryer.QueryRowContext(ctx, `
		SELECT id::text, inspector_id::text, result, notes, evidence_ids, inspected_at
		FROM site_inspections
		WHERE qualification_id = $1
		ORDER BY inspected_at DESC, id DESC
		LIMIT 1
	`, id).Scan(&inspection.ID, &inspection.InspectorID, &inspectionResult,
		&inspectionNotes, &inspectionEvidence, &inspection.InspectedAt)
	if err == nil {
		inspectionFound = true
		inspection.QualificationID = id
		inspection.Result = SiteInspectionResult(inspectionResult)
		inspection.Notes = inspectionNotes.String
		if err := unmarshalSafetyJSON(inspectionEvidence, &inspection.EvidenceIDs); err != nil {
			return MerchantQualification{}, err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return MerchantQualification{}, normalizeSafetyDBError(err)
	}
	if inspectionFound {
		qualification.SiteInspection = &inspection
	}
	qualification.EvidenceIDs, err = s.loadEvidenceIDs(ctx, queryer, EvidenceSubject{Type: EvidenceQualification, ID: id})
	if err != nil {
		return MerchantQualification{}, err
	}
	return qualification, nil
}

func (s *PostgresService) loadBatch(ctx context.Context, queryer safetyQueryer, id string, lock bool) (FoodBatch, error) {
	if !isSafetyUUID(id) {
		return FoodBatch{}, validationError(FieldError{"id", "must be a UUID"})
	}
	query := `
		SELECT id::text, merchant_id::text, product_id::text, campaign_id::text,
		       production_date, produced_at, expires_at, shelf_life_minutes,
		       storage_condition, quantity_planned, quantity_produced,
		       quantity_remaining, unit_weight_grams, specification,
		       ingredient_lots, status, created_by::text, created_at, updated_at
		FROM food_batches
		WHERE id = $1
	`
	if lock {
		query += " FOR UPDATE"
	}
	var batch FoodBatch
	var productID, campaignID sql.NullString
	var producedAt, expiresAt sql.NullTime
	var specification, ingredientLots []byte
	err := queryer.QueryRowContext(ctx, query, id).Scan(
		&batch.ID, &batch.MerchantID, &productID, &campaignID,
		&batch.ProductionDate, &producedAt, &expiresAt, &batch.ShelfLifeMinutes,
		&batch.StorageCondition, &batch.QuantityPlanned, &batch.QuantityProduced,
		&batch.QuantityRemaining, &batch.UnitWeightGrams, &specification,
		&ingredientLots, &batch.Status, &batch.CreatedBy, &batch.CreatedAt,
		&batch.UpdatedAt,
	)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	if productID.Valid {
		batch.ProductID = productID.String
	}
	if campaignID.Valid {
		batch.CampaignID = campaignID.String
	}
	batch.ProducedAt = nullTimePointer(producedAt)
	batch.ExpiresAt = nullTimePointer(expiresAt)
	if err := unmarshalSafetyJSON(specification, &batch.Specification); err != nil {
		return FoodBatch{}, err
	}
	if err := unmarshalSafetyJSON(ingredientLots, &batch.IngredientLots); err != nil {
		return FoodBatch{}, err
	}
	rows, err := queryer.QueryContext(ctx, `
		SELECT order_id::text, quantity, linked_by::text, linked_at
		FROM batch_order_associations
		WHERE batch_id = $1
		ORDER BY linked_at, order_id
	`, id)
	if err != nil {
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	for rows.Next() {
		var order OrderAssociation
		if err := rows.Scan(&order.OrderID, &order.Quantity, &order.LinkedBy, &order.LinkedAt); err != nil {
			rows.Close()
			return FoodBatch{}, normalizeSafetyDBError(err)
		}
		batch.Orders = append(batch.Orders, order)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return FoodBatch{}, normalizeSafetyDBError(err)
	}
	rows.Close()
	batch.EvidenceIDs, err = s.loadEvidenceIDs(ctx, queryer, EvidenceSubject{Type: EvidenceBatch, ID: id})
	if err != nil {
		return FoodBatch{}, err
	}
	return batch, nil
}

func (s *PostgresService) loadIncident(ctx context.Context, queryer safetyQueryer, id string, lock bool) (FoodSafetyIncident, error) {
	if !isSafetyUUID(id) {
		return FoodSafetyIncident{}, validationError(FieldError{"id", "must be a UUID"})
	}
	query := `
		SELECT id::text, merchant_id::text, reported_by::text, category, severity,
		       title, description, batch_ids, order_ids, status,
		       containment_action, investigation_summary, resolution_summary,
		       regulatory_report, reported_at, contained_at, resolved_at,
		       closed_at, created_at, updated_at
		FROM safety_incidents
		WHERE id = $1
	`
	if lock {
		query += " FOR UPDATE"
	}
	var incident FoodSafetyIncident
	var batchIDs, orderIDs, regulatoryReport []byte
	var containmentAction, investigationSummary, resolutionSummary sql.NullString
	var containedAt, resolvedAt, closedAt sql.NullTime
	err := queryer.QueryRowContext(ctx, query, id).Scan(
		&incident.ID, &incident.MerchantID, &incident.ReportedBy,
		&incident.Category, &incident.Severity, &incident.Title,
		&incident.Description, &batchIDs, &orderIDs, &incident.Status,
		&containmentAction, &investigationSummary, &resolutionSummary,
		&regulatoryReport, &incident.ReportedAt, &containedAt, &resolvedAt,
		&closedAt, &incident.CreatedAt, &incident.UpdatedAt,
	)
	if err != nil {
		return FoodSafetyIncident{}, normalizeSafetyDBError(err)
	}
	if err := unmarshalSafetyJSON(batchIDs, &incident.BatchIDs); err != nil {
		return FoodSafetyIncident{}, err
	}
	if err := unmarshalSafetyJSON(orderIDs, &incident.OrderIDs); err != nil {
		return FoodSafetyIncident{}, err
	}
	if err := unmarshalSafetyJSON(regulatoryReport, &incident.RegulatoryReport); err != nil {
		return FoodSafetyIncident{}, err
	}
	incident.ContainmentAction = containmentAction.String
	incident.InvestigationSummary = investigationSummary.String
	incident.ResolutionSummary = resolutionSummary.String
	incident.ContainedAt = nullTimePointer(containedAt)
	incident.ResolvedAt = nullTimePointer(resolvedAt)
	incident.ClosedAt = nullTimePointer(closedAt)
	incident.EvidenceIDs, err = s.loadEvidenceIDs(ctx, queryer, EvidenceSubject{Type: EvidenceIncident, ID: id})
	if err != nil {
		return FoodSafetyIncident{}, err
	}
	return incident, nil
}

func (s *PostgresService) loadEvidenceIDs(ctx context.Context, queryer safetyQueryer, subject EvidenceSubject) ([]string, error) {
	rows, err := queryer.QueryContext(ctx, `
		SELECT id::text
		FROM safety_evidences
		WHERE subject_type = $1 AND subject_id = $2
		ORDER BY sequence_no
	`, subject.Type, subject.ID)
	if err != nil {
		return nil, normalizeSafetyDBError(err)
	}
	defer rows.Close()
	result := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, normalizeSafetyDBError(err)
		}
		result = append(result, id)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizeSafetyDBError(err)
	}
	return result, nil
}

func (s *PostgresService) validateEvidenceRefsTx(ctx context.Context, tx *sql.Tx, subject EvidenceSubject, ids []string) error {
	for _, id := range uniqueStrings(ids) {
		if err := ensureSafetyUUID(id, "evidence_ids"); err != nil {
			return err
		}
		var evidenceType, evidenceSubjectID string
		err := tx.QueryRowContext(ctx, `
			SELECT subject_type, subject_id::text
			FROM safety_evidences
			WHERE id = $1
		`, id).Scan(&evidenceType, &evidenceSubjectID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return normalizeSafetyDBError(err)
		}
		if EvidenceSubjectType(evidenceType) != subject.Type || evidenceSubjectID != subject.ID {
			return fmt.Errorf("%w: evidence %s belongs to another subject", ErrConflict, id)
		}
	}
	return nil
}

func (s *PostgresService) lockEvidenceSubjectTx(ctx context.Context, tx *sql.Tx, subject EvidenceSubject) error {
	switch subject.Type {
	case EvidenceQualification:
		return s.lockSubjectRow(ctx, tx, `SELECT 1 FROM merchant_qualifications WHERE id = $1 FOR UPDATE`, subject.ID)
	case EvidenceBatch:
		return s.lockSubjectRow(ctx, tx, `SELECT 1 FROM food_batches WHERE id = $1 FOR UPDATE`, subject.ID)
	case EvidenceIncident:
		return s.lockSubjectRow(ctx, tx, `SELECT 1 FROM safety_incidents WHERE id = $1 FOR UPDATE`, subject.ID)
	default:
		return fmt.Errorf("%w: unsupported evidence subject %s", ErrInvalidState, subject.Type)
	}
}

func (s *PostgresService) ensureEvidenceSubjectExists(ctx context.Context, queryer safetyQueryer, subject EvidenceSubject) error {
	var query string
	switch subject.Type {
	case EvidenceQualification:
		query = `SELECT 1 FROM merchant_qualifications WHERE id = $1`
	case EvidenceBatch:
		query = `SELECT 1 FROM food_batches WHERE id = $1`
	case EvidenceIncident:
		query = `SELECT 1 FROM safety_incidents WHERE id = $1`
	default:
		return fmt.Errorf("%w: unsupported evidence subject %s", ErrInvalidState, subject.Type)
	}
	return s.scanExists(ctx, queryer, query, subject.ID)
}

func (s *PostgresService) scanExists(ctx context.Context, queryer safetyQueryer, query string, id string) error {
	var exists int
	err := queryer.QueryRowContext(ctx, query, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return normalizeSafetyDBError(err)
}

func (s *PostgresService) lockSubjectRow(ctx context.Context, tx *sql.Tx, query string, id string) error {
	return s.scanExists(ctx, tx, query, id)
}

func (s *PostgresService) quarantineIncidentBatchesTx(ctx context.Context, tx *sql.Tx, incident FoodSafetyIncident) error {
	type batchState struct {
		id       string
		status   FoodBatchStatus
		produced int
	}
	states := make([]batchState, 0, len(incident.BatchIDs))
	for _, batchID := range incident.BatchIDs {
		var state batchState
		err := tx.QueryRowContext(ctx, `
			SELECT id::text, status, quantity_produced
			FROM food_batches
			WHERE id = $1 AND merchant_id = $2
			FOR UPDATE
		`, batchID, incident.MerchantID).Scan(&state.id, &state.status, &state.produced)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return normalizeSafetyDBError(err)
		}
		if state.status == BatchQuarantined || state.status == BatchRecalled ||
			state.status == BatchDisposed || state.status == BatchCancelled {
			states = append(states, state)
			continue
		}
		if !CanTransitionBatch(state.status, BatchQuarantined) {
			return fmt.Errorf("%w: cannot quarantine batch %s from %s", ErrInvalidTransition, state.id, state.status)
		}
		states = append(states, state)
	}
	now := s.currentTime()
	for _, state := range states {
		if state.status == BatchQuarantined || state.status == BatchRecalled ||
			state.status == BatchDisposed || state.status == BatchCancelled {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE food_batches SET status = $2, updated_at = $3 WHERE id = $1
		`, state.id, BatchQuarantined, now); err != nil {
			return normalizeSafetyDBError(err)
		}
	}
	return nil
}

func scanPersistedEvidence(scanner interface{ Scan(...any) error }, subject EvidenceSubject) (Evidence, error) {
	var evidence Evidence
	var metadata []byte
	var previousHash sql.NullString
	var capturedAt time.Time
	err := scanner.Scan(&evidence.ID, &evidence.Sequence, &previousHash,
		&evidence.Hash, &evidence.Kind, &evidence.URI, &evidence.SHA256,
		&metadata, &evidence.CapturedBy, &capturedAt)
	if err != nil {
		return Evidence{}, normalizeSafetyDBError(err)
	}
	evidence.Subject = subject
	evidence.PreviousHash = previousHash.String
	evidence.CapturedAt = capturedAt
	evidence.CreatedAt = capturedAt
	var internal persistedEvidenceMetadata
	if err := unmarshalPersistedEvidenceMetadata(metadata, &evidence, &internal); err != nil {
		return Evidence{}, err
	}
	if !internal.CreatedAt.IsZero() {
		evidence.CreatedAt = internal.CreatedAt
	}
	return evidence, nil
}

type persistedEvidenceMetadata struct {
	UserMetadata map[string]string `json:"user_metadata,omitempty"`
	MimeType     string            `json:"mime_type,omitempty"`
	Description  string            `json:"description,omitempty"`
	CreatedAt    time.Time         `json:"created_at,omitempty"`
}

func marshalPersistedEvidenceMetadata(evidence Evidence) ([]byte, error) {
	return json.Marshal(map[string]any{
		"__chihuo": persistedEvidenceMetadata{
			UserMetadata: evidence.Metadata,
			MimeType:     evidence.MimeType,
			Description:  evidence.Description,
			CreatedAt:    evidence.CreatedAt,
		},
	})
}

func unmarshalPersistedEvidenceMetadata(raw []byte, evidence *Evidence, internal *persistedEvidenceMetadata) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var envelope struct {
		Internal *persistedEvidenceMetadata `json:"__chihuo"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if envelope.Internal == nil {
		return json.Unmarshal(raw, &evidence.Metadata)
	}
	*internal = *envelope.Internal
	evidence.Metadata = cloneStringMap(internal.UserMetadata)
	evidence.MimeType = internal.MimeType
	evidence.Description = internal.Description
	return nil
}

func marshalSafetyJSON(value any, fallback any) ([]byte, error) {
	if value == nil {
		value = fallback
	}
	return json.Marshal(value)
}

func unmarshalSafetyJSON(raw []byte, target any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func ensureSafetyUUID(value, field string) error {
	if !isSafetyUUID(strings.TrimSpace(value)) {
		return validationError(FieldError{field, "must be a UUID"})
	}
	return nil
}

func isSafetyUUID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 36 {
		return false
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return false
			}
			continue
		}
		if !((character >= '0' && character <= '9') ||
			(character >= 'a' && character <= 'f') ||
			(character >= 'A' && character <= 'F')) {
			return false
		}
	}
	return true
}

func newSafetyUUID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("00000000-0000-4000-8000-%012d", time.Now().UnixNano()%1000000000000)
	}
	buffer[6] = (buffer[6] & 0x0f) | 0x40
	buffer[8] = (buffer[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(buffer[0:4]),
		hex.EncodeToString(buffer[4:6]),
		hex.EncodeToString(buffer[6:8]),
		hex.EncodeToString(buffer[8:10]),
		hex.EncodeToString(buffer[10:16]))
}

func normalizeSafetyDBError(err error) error {
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
			return fmt.Errorf("%w: %s", ErrConflict, pgErr.ConstraintName)
		case "23503":
			return fmt.Errorf("%w: %s", ErrConflict, pgErr.ConstraintName)
		case "23514":
			return fmt.Errorf("%w: %s", ErrInvalidState, pgErr.ConstraintName)
		}
	}
	return err
}
