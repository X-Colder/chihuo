package safety

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var sha256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func ValidateQualification(q MerchantQualification, now time.Time) error {
	fields := make([]FieldError, 0)
	if strings.TrimSpace(q.ID) == "" {
		fields = append(fields, FieldError{"id", "is required"})
	}
	if strings.TrimSpace(q.MerchantID) == "" {
		fields = append(fields, FieldError{"merchant_id", "is required"})
	}
	if !q.Status.Valid() {
		fields = append(fields, FieldError{"status", "is invalid"})
	}
	if q.BusinessLicenseIssuedAt.IsZero() || q.BusinessLicenseExpiresAt.IsZero() ||
		!q.BusinessLicenseExpiresAt.After(q.BusinessLicenseIssuedAt) {
		fields = append(fields, FieldError{"business_license_dates", "issue and expiry dates are invalid"})
	}
	if q.FoodPermitIssuedAt.IsZero() || q.FoodPermitExpiresAt.IsZero() ||
		!q.FoodPermitExpiresAt.After(q.FoodPermitIssuedAt) {
		fields = append(fields, FieldError{"food_permit_dates", "issue and expiry dates are invalid"})
	}
	if q.Status == QualificationApproved {
		if !q.BusinessLicenseExpiresAt.After(now) {
			fields = append(fields, FieldError{"business_license_expires_at", "must be in the future for an approved qualification"})
		}
		if !q.FoodPermitExpiresAt.After(now) {
			fields = append(fields, FieldError{"food_permit_expires_at", "must be in the future for an approved qualification"})
		}
	}
	for i, document := range q.Documents {
		prefix := fmt.Sprintf("documents[%d]", i)
		if strings.TrimSpace(document.ID) == "" {
			fields = append(fields, FieldError{prefix + ".id", "is required"})
		}
		if !document.Kind.Valid() {
			fields = append(fields, FieldError{prefix + ".kind", "is invalid"})
		}
		if err := validateEvidencePayload(document.URI, document.SHA256); err != nil {
			fields = append(fields, FieldError{prefix, err.Error()})
		}
	}
	if q.SiteInspection != nil {
		if q.SiteInspection.QualificationID != q.ID {
			fields = append(fields, FieldError{"site_inspection.qualification_id", "must match qualification id"})
		}
		if !q.SiteInspection.Result.Valid() {
			fields = append(fields, FieldError{"site_inspection.result", "is invalid"})
		}
	}
	if len(fields) > 0 {
		return validationError(fields...)
	}
	return nil
}

func ValidateQualificationForSubmission(q MerchantQualification, now time.Time) error {
	if err := ValidateQualification(q, now); err != nil {
		return err
	}
	fields := make([]FieldError, 0)
	required := map[string]string{
		"legal_entity_name":       q.LegalEntityName,
		"store_name":              q.StoreName,
		"business_license_number": q.BusinessLicenseNumber,
		"food_permit_number":      q.FoodPermitNumber,
		"registered_address":      q.RegisteredAddress,
		"operating_address":       q.OperatingAddress,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			fields = append(fields, FieldError{field, "is required before submission"})
		}
	}
	if !q.BusinessLicenseExpiresAt.After(now) {
		fields = append(fields, FieldError{"business_license_expires_at", "must be in the future"})
	}
	if !q.FoodPermitExpiresAt.After(now) {
		fields = append(fields, FieldError{"food_permit_expires_at", "must be in the future"})
	}
	documentKinds := map[QualificationDocumentKind]bool{}
	for _, document := range q.Documents {
		documentKinds[document.Kind] = true
	}
	if !documentKinds[DocumentBusinessLicense] {
		fields = append(fields, FieldError{"documents", "business license document is required"})
	}
	if !documentKinds[DocumentFoodPermit] {
		fields = append(fields, FieldError{"documents", "food permit document is required"})
	}
	if len(fields) > 0 {
		return validationError(fields...)
	}
	return nil
}

func ValidateBatch(batch FoodBatch, now time.Time) error {
	fields := make([]FieldError, 0)
	if strings.TrimSpace(batch.ID) == "" {
		fields = append(fields, FieldError{"id", "is required"})
	}
	if strings.TrimSpace(batch.MerchantID) == "" {
		fields = append(fields, FieldError{"merchant_id", "is required"})
	}
	if strings.TrimSpace(batch.ProductID) == "" {
		fields = append(fields, FieldError{"product_id", "is required"})
	}
	if strings.TrimSpace(batch.StorageCondition) == "" {
		fields = append(fields, FieldError{"storage_condition", "is required"})
	}
	if batch.ProductionDate.IsZero() {
		fields = append(fields, FieldError{"production_date", "is required"})
	}
	if batch.ShelfLifeMinutes <= 0 {
		fields = append(fields, FieldError{"shelf_life_minutes", "must be positive"})
	}
	if batch.QuantityPlanned <= 0 || batch.QuantityProduced < 0 || batch.QuantityProduced > batch.QuantityPlanned {
		fields = append(fields, FieldError{"quantity", "planned and produced quantities are inconsistent"})
	}
	if batch.QuantityRemaining < 0 || batch.QuantityRemaining > batch.QuantityProduced {
		fields = append(fields, FieldError{"quantity_remaining", "must be between zero and quantity produced"})
	}
	if batch.UnitWeightGrams <= 0 {
		fields = append(fields, FieldError{"unit_weight_grams", "must be positive"})
	}
	if batch.Specification.WeightGrams <= 0 || batch.Specification.WeightGrams != batch.UnitWeightGrams {
		fields = append(fields, FieldError{"specification.weight_grams", "must be positive and match unit_weight_grams"})
	}
	if len(batch.Specification.Ingredients) == 0 {
		fields = append(fields, FieldError{"specification.ingredients", "at least one ingredient is required"})
	}
	if batch.Status != BatchDraft && batch.Status != BatchCancelled &&
		batch.ProducedAt == nil {
		fields = append(fields, FieldError{"produced_at", "is required after production starts"})
	}
	if batch.ProducedAt != nil && batch.ExpiresAt != nil && !batch.ExpiresAt.After(*batch.ProducedAt) {
		fields = append(fields, FieldError{"expires_at", "must be after produced_at"})
	}
	if batch.ExpiresAt != nil && !batch.ExpiresAt.After(now) &&
		batch.Status != BatchQuarantined && batch.Status != BatchRecalled &&
		batch.Status != BatchDisposed && batch.Status != BatchCancelled {
		fields = append(fields, FieldError{"expires_at", "must be in the future for an active batch"})
	}
	for i, lot := range batch.IngredientLots {
		prefix := fmt.Sprintf("ingredient_lots[%d]", i)
		for field, value := range map[string]string{
			"ingredient": lot.Ingredient,
			"supplier":   lot.Supplier,
			"lot_number": lot.LotNumber,
		} {
			if strings.TrimSpace(value) == "" {
				fields = append(fields, FieldError{prefix + "." + field, "is required"})
			}
		}
		if lot.ExpiresAt.IsZero() || !lot.ExpiresAt.After(lot.ReceivedAt) {
			fields = append(fields, FieldError{prefix + ".expires_at", "must be after received_at"})
		}
		if lot.QuantityGrams <= 0 {
			fields = append(fields, FieldError{prefix + ".quantity_grams", "must be positive"})
		}
	}
	for i, order := range batch.Orders {
		prefix := fmt.Sprintf("orders[%d]", i)
		if strings.TrimSpace(order.OrderID) == "" || order.Quantity <= 0 || strings.TrimSpace(order.LinkedBy) == "" {
			fields = append(fields, FieldError{prefix, "order id, quantity and linked_by are required"})
		}
	}
	if len(fields) > 0 {
		return validationError(fields...)
	}
	return nil
}

func ValidateIncident(incident FoodSafetyIncident) error {
	fields := make([]FieldError, 0)
	if strings.TrimSpace(incident.ID) == "" {
		fields = append(fields, FieldError{"id", "is required"})
	}
	if strings.TrimSpace(incident.MerchantID) == "" {
		fields = append(fields, FieldError{"merchant_id", "is required"})
	}
	if strings.TrimSpace(incident.ReportedBy) == "" {
		fields = append(fields, FieldError{"reported_by", "is required"})
	}
	if strings.TrimSpace(incident.Category) == "" {
		fields = append(fields, FieldError{"category", "is required"})
	}
	if !incident.Severity.Valid() {
		fields = append(fields, FieldError{"severity", "is invalid"})
	}
	if strings.TrimSpace(incident.Title) == "" || strings.TrimSpace(incident.Description) == "" {
		fields = append(fields, FieldError{"description", "title and description are required"})
	}
	if len(incident.BatchIDs) == 0 && len(incident.OrderIDs) == 0 {
		fields = append(fields, FieldError{"affected_resources", "at least one batch_id or order_id is required"})
	}
	if !incident.Status.Valid() {
		fields = append(fields, FieldError{"status", "is invalid"})
	}
	if incident.ReportedAt.IsZero() {
		fields = append(fields, FieldError{"reported_at", "is required"})
	}
	if incident.RegulatoryReport.Reported && incident.RegulatoryReport.ReportedAt == nil {
		fields = append(fields, FieldError{"regulatory_report.reported_at", "is required when reported is true"})
	}
	if len(fields) > 0 {
		return validationError(fields...)
	}
	return nil
}

func ValidateRecall(recall Recall) error {
	fields := make([]FieldError, 0)
	if strings.TrimSpace(recall.ID) == "" || strings.TrimSpace(recall.IncidentID) == "" ||
		strings.TrimSpace(recall.MerchantID) == "" {
		fields = append(fields, FieldError{"identity", "id, incident_id and merchant_id are required"})
	}
	if !recall.Scope.Valid() {
		fields = append(fields, FieldError{"scope", "is invalid"})
	}
	if strings.TrimSpace(recall.Reason) == "" {
		fields = append(fields, FieldError{"reason", "is required"})
	}
	if len(recall.BatchIDs) == 0 && len(recall.OrderIDs) == 0 {
		fields = append(fields, FieldError{"affected_resources", "at least one batch_id or order_id is required"})
	}
	if recall.AffectedQuantity <= 0 || recall.RecoveredQuantity < 0 ||
		recall.DisposedQuantity < 0 ||
		recall.RecoveredQuantity+recall.DisposedQuantity > recall.AffectedQuantity {
		fields = append(fields, FieldError{"quantities", "affected, recovered and disposed quantities are inconsistent"})
	}
	if !recall.Status.Valid() {
		fields = append(fields, FieldError{"status", "is invalid"})
	}
	if len(fields) > 0 {
		return validationError(fields...)
	}
	return nil
}

func ValidateDisposition(disposition Disposition) error {
	fields := make([]FieldError, 0)
	if strings.TrimSpace(disposition.ID) == "" || strings.TrimSpace(disposition.IncidentID) == "" {
		fields = append(fields, FieldError{"identity", "id and incident_id are required"})
	}
	if strings.TrimSpace(disposition.BatchID) == "" && strings.TrimSpace(disposition.OrderID) == "" {
		fields = append(fields, FieldError{"target", "batch_id or order_id is required"})
	}
	if !disposition.Type.Valid() {
		fields = append(fields, FieldError{"type", "is invalid"})
	}
	if !disposition.Status.Valid() {
		fields = append(fields, FieldError{"status", "is invalid"})
	}
	if disposition.Quantity <= 0 {
		fields = append(fields, FieldError{"quantity", "must be positive"})
	}
	if strings.TrimSpace(disposition.Unit) == "" || strings.TrimSpace(disposition.ActionBy) == "" {
		fields = append(fields, FieldError{"action", "unit and action_by are required"})
	}
	if len(fields) > 0 {
		return validationError(fields...)
	}
	return nil
}

func ValidateEvidence(e Evidence) error {
	fields := make([]FieldError, 0)
	if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.Subject.ID) == "" {
		fields = append(fields, FieldError{"identity", "id and subject id are required"})
	}
	if !e.Subject.Type.Valid() {
		fields = append(fields, FieldError{"subject.type", "is invalid"})
	}
	if !e.Kind.Valid() {
		fields = append(fields, FieldError{"kind", "is invalid"})
	}
	if err := validateEvidencePayload(e.URI, e.SHA256); err != nil {
		fields = append(fields, FieldError{"payload", err.Error()})
	}
	if strings.TrimSpace(e.CapturedBy) == "" || e.CapturedAt.IsZero() {
		fields = append(fields, FieldError{"capture", "captured_by and captured_at are required"})
	}
	if e.Sequence == 0 {
		fields = append(fields, FieldError{"sequence", "must start at one"})
	}
	if !sha256Pattern.MatchString(e.Hash) {
		fields = append(fields, FieldError{"hash", "must be a SHA-256 hex digest"})
	}
	if e.Sequence == 1 && e.PreviousHash != "" {
		fields = append(fields, FieldError{"previous_hash", "must be empty for the first evidence record"})
	}
	if e.Sequence > 1 && !sha256Pattern.MatchString(e.PreviousHash) {
		fields = append(fields, FieldError{"previous_hash", "must be a SHA-256 hex digest after the first record"})
	}
	if len(fields) > 0 {
		return validationError(fields...)
	}
	return nil
}

func validateEvidencePayload(uri, digest string) error {
	if strings.TrimSpace(uri) == "" {
		return fmt.Errorf("uri is required")
	}
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("uri must include a scheme")
	}
	if !sha256Pattern.MatchString(digest) {
		return fmt.Errorf("sha256 must be a SHA-256 hex digest")
	}
	return nil
}

func evidenceHash(e Evidence) string {
	payload := struct {
		Subject      EvidenceSubject   `json:"subject"`
		Sequence     uint64            `json:"sequence"`
		PreviousHash string            `json:"previous_hash,omitempty"`
		Kind         EvidenceKind      `json:"kind"`
		URI          string            `json:"uri"`
		SHA256       string            `json:"sha256"`
		MimeType     string            `json:"mime_type,omitempty"`
		CapturedBy   string            `json:"captured_by"`
		CapturedAt   string            `json:"captured_at"`
		Description  string            `json:"description,omitempty"`
		Metadata     map[string]string `json:"metadata,omitempty"`
	}{
		Subject:      e.Subject,
		Sequence:     e.Sequence,
		PreviousHash: e.PreviousHash,
		Kind:         e.Kind,
		URI:          e.URI,
		SHA256:       strings.ToLower(e.SHA256),
		MimeType:     e.MimeType,
		CapturedBy:   e.CapturedBy,
		CapturedAt:   e.CapturedAt.UTC().Format(time.RFC3339Nano),
		Description:  e.Description,
		Metadata:     e.Metadata,
	}
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
