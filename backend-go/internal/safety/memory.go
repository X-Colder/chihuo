package safety

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryService struct {
	mu sync.RWMutex

	now func() time.Time

	qualifications map[string]MerchantQualification
	batches        map[string]FoodBatch
	incidents      map[string]FoodSafetyIncident
	recalls        map[string]Recall
	dispositions   map[string]Disposition
	evidence       map[string]Evidence
	evidenceChains map[string][]string
	orderBatch     map[string]string
}

func NewMemoryService() *MemoryService {
	return NewMemoryServiceWithClock(nil)
}

func NewMemoryServiceWithClock(now func() time.Time) *MemoryService {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &MemoryService{
		now:            now,
		qualifications: make(map[string]MerchantQualification),
		batches:        make(map[string]FoodBatch),
		incidents:      make(map[string]FoodSafetyIncident),
		recalls:        make(map[string]Recall),
		dispositions:   make(map[string]Disposition),
		evidence:       make(map[string]Evidence),
		evidenceChains: make(map[string][]string),
		orderBatch:     make(map[string]string),
	}
}

func (s *MemoryService) currentTime() time.Time {
	return s.now().UTC()
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (s *MemoryService) CreateQualification(ctx context.Context, input CreateQualificationInput) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return MerchantQualification{}, err
	}
	now := s.currentTime()
	qualification := MerchantQualification{
		ID:                       newID("qualification"),
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
		BusinessScope:            cloneStrings(input.BusinessScope),
		Documents:                cloneDocuments(input.Documents),
		Status:                   QualificationDraft,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	for i := range qualification.Documents {
		if strings.TrimSpace(qualification.Documents[i].ID) == "" {
			qualification.Documents[i].ID = newID("document")
		}
		if qualification.Documents[i].UploadedAt.IsZero() {
			qualification.Documents[i].UploadedAt = now
		}
		if strings.TrimSpace(qualification.Documents[i].UploadedBy) == "" {
			qualification.Documents[i].UploadedBy = strings.TrimSpace(input.CreatedBy)
		}
	}
	if err := ValidateQualification(qualification, now); err != nil {
		return MerchantQualification{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.qualifications[qualification.ID] = cloneQualification(qualification)
	return cloneQualification(qualification), nil
}

func (s *MemoryService) GetQualification(ctx context.Context, id string) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return MerchantQualification{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	qualification, ok := s.qualifications[id]
	if !ok {
		return MerchantQualification{}, ErrNotFound
	}
	return cloneQualification(qualification), nil
}

func (s *MemoryService) ListQualifications(ctx context.Context, merchantID string, status QualificationStatus) ([]MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]MerchantQualification, 0)
	for _, qualification := range s.qualifications {
		if merchantID != "" && qualification.MerchantID != merchantID {
			continue
		}
		if status != "" && qualification.Status != status {
			continue
		}
		result = append(result, cloneQualification(qualification))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *MemoryService) SubmitQualification(ctx context.Context, id string, input SubmitQualificationInput) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return MerchantQualification{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return MerchantQualification{}, validationError(FieldError{"actor_id", "is required"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	qualification, ok := s.qualifications[id]
	if !ok {
		return MerchantQualification{}, ErrNotFound
	}
	if !CanTransitionQualification(qualification.Status, QualificationPendingReview) {
		return MerchantQualification{}, invalidTransition(string(qualification.Status), string(QualificationPendingReview))
	}
	if err := ValidateQualificationForSubmission(qualification, s.currentTime()); err != nil {
		return MerchantQualification{}, err
	}
	if err := s.validateEvidenceRefsLocked(EvidenceSubject{Type: EvidenceQualification, ID: id}, input.EvidenceIDs); err != nil {
		return MerchantQualification{}, err
	}
	now := s.currentTime()
	from := qualification.Status
	qualification.Status = QualificationPendingReview
	qualification.SubmittedAt = timePtr(now)
	qualification.UpdatedAt = now
	qualification.EvidenceIDs = appendUnique(qualification.EvidenceIDs, input.EvidenceIDs...)
	appendStatusChange(&qualification.StatusHistory, string(from), string(QualificationPendingReview), input.ActorID, "submitted", now)
	s.qualifications[id] = qualification
	return cloneQualification(qualification), nil
}

func (s *MemoryService) ReviewQualification(ctx context.Context, id string, input ReviewQualificationInput) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
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
	s.mu.Lock()
	defer s.mu.Unlock()
	qualification, ok := s.qualifications[id]
	if !ok {
		return MerchantQualification{}, ErrNotFound
	}
	if qualification.Status != QualificationPendingReview {
		return MerchantQualification{}, invalidTransition(string(qualification.Status), string(input.Status))
	}
	if err := s.validateEvidenceRefsLocked(EvidenceSubject{Type: EvidenceQualification, ID: id}, input.EvidenceIDs); err != nil {
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
	from := qualification.Status
	qualification.Status = input.Status
	qualification.ReviewedAt = timePtr(now)
	qualification.UpdatedAt = now
	qualification.EvidenceIDs = appendUnique(qualification.EvidenceIDs, input.EvidenceIDs...)
	review := QualificationReview{
		ID:              newID("review"),
		QualificationID: id,
		ReviewerID:      input.ReviewerID,
		FromStatus:      from,
		ToStatus:        input.Status,
		Reason:          strings.TrimSpace(input.Reason),
		EvidenceIDs:     cloneStrings(input.EvidenceIDs),
		ReviewedAt:      now,
	}
	qualification.ReviewHistory = append(qualification.ReviewHistory, review)
	appendStatusChange(&qualification.StatusHistory, string(from), string(input.Status), input.ReviewerID, input.Reason, now)
	s.qualifications[id] = qualification
	return cloneQualification(qualification), nil
}

func (s *MemoryService) RecordSiteInspection(ctx context.Context, id string, input SiteInspectionInput) (MerchantQualification, error) {
	if err := checkContext(ctx); err != nil {
		return MerchantQualification{}, err
	}
	if strings.TrimSpace(input.InspectorID) == "" {
		return MerchantQualification{}, validationError(FieldError{"inspector_id", "is required"})
	}
	if !input.Result.Valid() {
		return MerchantQualification{}, validationError(FieldError{"result", "is invalid"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	qualification, ok := s.qualifications[id]
	if !ok {
		return MerchantQualification{}, ErrNotFound
	}
	if qualification.Status != QualificationDraft && qualification.Status != QualificationPendingReview {
		return MerchantQualification{}, fmt.Errorf("%w: inspection is not allowed in %s", ErrInvalidState, qualification.Status)
	}
	if err := s.validateEvidenceRefsLocked(EvidenceSubject{Type: EvidenceQualification, ID: id}, input.EvidenceIDs); err != nil {
		return MerchantQualification{}, err
	}
	now := s.currentTime()
	inspectedAt := input.InspectedAt
	if inspectedAt.IsZero() {
		inspectedAt = now
	}
	qualification.SiteInspection = &SiteInspection{
		ID:              newID("inspection"),
		QualificationID: id,
		InspectorID:     input.InspectorID,
		Result:          input.Result,
		Notes:           strings.TrimSpace(input.Notes),
		EvidenceIDs:     cloneStrings(input.EvidenceIDs),
		InspectedAt:     inspectedAt,
	}
	qualification.EvidenceIDs = appendUnique(qualification.EvidenceIDs, input.EvidenceIDs...)
	qualification.UpdatedAt = now
	s.qualifications[id] = qualification
	return cloneQualification(qualification), nil
}

func (s *MemoryService) CreateBatch(ctx context.Context, input CreateBatchInput) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	if strings.TrimSpace(input.CreatedBy) == "" {
		return FoodBatch{}, validationError(FieldError{"created_by", "is required"})
	}
	now := s.currentTime()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.approvedQualificationLocked(input.MerchantID, now); !ok {
		return FoodBatch{}, fmt.Errorf("%w: merchant qualification is not approved and current", ErrInvalidState)
	}
	batch := FoodBatch{
		ID:               newID("batch"),
		MerchantID:       strings.TrimSpace(input.MerchantID),
		ProductID:        strings.TrimSpace(input.ProductID),
		CampaignID:       strings.TrimSpace(input.CampaignID),
		ProductionDate:   input.ProductionDate,
		ShelfLifeMinutes: input.ShelfLifeMinutes,
		StorageCondition: strings.TrimSpace(input.StorageCondition),
		QuantityPlanned:  input.QuantityPlanned,
		UnitWeightGrams:  input.UnitWeightGrams,
		Specification:    cloneProductSpec(input.Specification),
		IngredientLots:   cloneIngredientLots(input.IngredientLots),
		Status:           BatchDraft,
		CreatedBy:        strings.TrimSpace(input.CreatedBy),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := ValidateBatch(batch, now); err != nil {
		return FoodBatch{}, err
	}
	s.batches[batch.ID] = cloneBatch(batch)
	return cloneBatch(batch), nil
}

func (s *MemoryService) GetBatch(ctx context.Context, id string) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	batch, ok := s.batches[id]
	if !ok {
		return FoodBatch{}, ErrNotFound
	}
	return cloneBatch(batch), nil
}

func (s *MemoryService) ListBatches(ctx context.Context, merchantID string, status FoodBatchStatus) ([]FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]FoodBatch, 0)
	for _, batch := range s.batches {
		if merchantID != "" && batch.MerchantID != merchantID {
			continue
		}
		if status != "" && batch.Status != status {
			continue
		}
		result = append(result, cloneBatch(batch))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *MemoryService) RecordBatchProduction(ctx context.Context, id string, input RecordProductionInput) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return FoodBatch{}, validationError(FieldError{"actor_id", "is required"})
	}
	if input.Quantity <= 0 {
		return FoodBatch{}, validationError(FieldError{"quantity", "must be positive"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	batch, ok := s.batches[id]
	if !ok {
		return FoodBatch{}, ErrNotFound
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
	if expiresAt.IsZero() {
		shelfLife := batch.ShelfLifeMinutes
		if input.ShelfLifeMinutes > 0 {
			shelfLife = input.ShelfLifeMinutes
		}
		expiresAt = producedAt.Add(time.Duration(shelfLife) * time.Minute)
	}
	if !expiresAt.After(producedAt) {
		return FoodBatch{}, validationError(FieldError{"expires_at", "must be after produced_at"})
	}
	if input.ShelfLifeMinutes > 0 {
		batch.ShelfLifeMinutes = input.ShelfLifeMinutes
	}
	batch.QuantityProduced = input.Quantity
	batch.QuantityRemaining = input.Quantity
	batch.ProducedAt = timePtr(producedAt)
	batch.ExpiresAt = timePtr(expiresAt)
	batch.UpdatedAt = s.currentTime()
	s.batches[id] = batch
	return cloneBatch(batch), nil
}

func (s *MemoryService) AssociateOrders(ctx context.Context, id string, input AssociateOrdersInput) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return FoodBatch{}, validationError(FieldError{"actor_id", "is required"})
	}
	if len(input.Orders) == 0 {
		return FoodBatch{}, validationError(FieldError{"orders", "at least one order is required"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	batch, ok := s.batches[id]
	if !ok {
		return FoodBatch{}, ErrNotFound
	}
	switch batch.Status {
	case BatchDraft, BatchScheduled, BatchProducing, BatchPacked, BatchReadyForHandoff:
	default:
		return FoodBatch{}, fmt.Errorf("%w: orders cannot be associated with %s batch", ErrInvalidState, batch.Status)
	}
	existingOrders := make(map[string]bool, len(batch.Orders))
	totalQuantity := 0
	for _, order := range batch.Orders {
		existingOrders[order.OrderID] = true
		totalQuantity += order.Quantity
	}
	addedOrders := make(map[string]bool, len(input.Orders))
	now := s.currentTime()
	for _, inputOrder := range input.Orders {
		orderID := strings.TrimSpace(inputOrder.OrderID)
		if orderID == "" || inputOrder.Quantity <= 0 {
			return FoodBatch{}, validationError(FieldError{"orders", "order_id and quantity are required"})
		}
		if existingOrders[orderID] || addedOrders[orderID] {
			return FoodBatch{}, fmt.Errorf("%w: order %s is already associated", ErrConflict, orderID)
		}
		if assignedBatchID, exists := s.orderBatch[orderID]; exists {
			return FoodBatch{}, fmt.Errorf("%w: order %s already belongs to batch %s", ErrConflict, orderID, assignedBatchID)
		}
		if totalQuantity+inputOrder.Quantity > batch.QuantityPlanned {
			return FoodBatch{}, fmt.Errorf("%w: associated order quantity exceeds planned batch quantity", ErrConflict)
		}
		addedOrders[orderID] = true
		totalQuantity += inputOrder.Quantity
		batch.Orders = append(batch.Orders, OrderAssociation{
			OrderID:  orderID,
			Quantity: inputOrder.Quantity,
			LinkedBy: input.ActorID,
			LinkedAt: now,
		})
	}
	for orderID := range addedOrders {
		s.orderBatch[orderID] = id
	}
	batch.UpdatedAt = now
	s.batches[id] = batch
	return cloneBatch(batch), nil
}

func (s *MemoryService) TransitionBatch(ctx context.Context, id string, target FoodBatchStatus, input TransitionInput) (FoodBatch, error) {
	if err := checkContext(ctx); err != nil {
		return FoodBatch{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return FoodBatch{}, validationError(FieldError{"actor_id", "is required"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	batch, ok := s.batches[id]
	if !ok {
		return FoodBatch{}, ErrNotFound
	}
	if !target.Valid() {
		return FoodBatch{}, validationError(FieldError{"status", "is invalid"})
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
	if err := s.validateEvidenceRefsLocked(EvidenceSubject{Type: EvidenceBatch, ID: id}, input.EvidenceIDs); err != nil {
		return FoodBatch{}, err
	}
	now := s.currentTime()
	from := batch.Status
	batch.Status = target
	batch.EvidenceIDs = appendUnique(batch.EvidenceIDs, input.EvidenceIDs...)
	batch.UpdatedAt = now
	appendStatusChange(&batch.StatusHistory, string(from), string(target), input.ActorID, input.Reason, now)
	s.batches[id] = batch
	return cloneBatch(batch), nil
}

func (s *MemoryService) CreateIncident(ctx context.Context, input CreateIncidentInput) (FoodSafetyIncident, error) {
	if err := checkContext(ctx); err != nil {
		return FoodSafetyIncident{}, err
	}
	if len(input.EvidenceIDs) > 0 {
		return FoodSafetyIncident{}, fmt.Errorf("%w: add incident evidence after creation", ErrConflict)
	}
	now := s.currentTime()
	reportedAt := input.ReportedAt
	if reportedAt.IsZero() {
		reportedAt = now
	}
	incident := FoodSafetyIncident{
		ID:               newID("incident"),
		MerchantID:       strings.TrimSpace(input.MerchantID),
		ReportedBy:       strings.TrimSpace(input.ReportedBy),
		Category:         strings.TrimSpace(input.Category),
		Severity:         input.Severity,
		Title:            strings.TrimSpace(input.Title),
		Description:      strings.TrimSpace(input.Description),
		BatchIDs:         uniqueStrings(input.BatchIDs),
		OrderIDs:         uniqueStrings(input.OrderIDs),
		Status:           IncidentReported,
		RegulatoryReport: cloneRegulatoryReport(input.RegulatoryReport),
		ReportedAt:       reportedAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := ValidateIncident(incident); err != nil {
		return FoodSafetyIncident{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.validateIncidentBatchesLocked(incident.MerchantID, incident.BatchIDs); err != nil {
		return FoodSafetyIncident{}, err
	}
	s.incidents[incident.ID] = cloneIncident(incident)
	return cloneIncident(incident), nil
}

func (s *MemoryService) GetIncident(ctx context.Context, id string) (FoodSafetyIncident, error) {
	if err := checkContext(ctx); err != nil {
		return FoodSafetyIncident{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	incident, ok := s.incidents[id]
	if !ok {
		return FoodSafetyIncident{}, ErrNotFound
	}
	return cloneIncident(incident), nil
}

func (s *MemoryService) ListIncidents(ctx context.Context, merchantID string, status IncidentStatus) ([]FoodSafetyIncident, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]FoodSafetyIncident, 0)
	for _, incident := range s.incidents {
		if merchantID != "" && incident.MerchantID != merchantID {
			continue
		}
		if status != "" && incident.Status != status {
			continue
		}
		result = append(result, cloneIncident(incident))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ReportedAt.After(result[j].ReportedAt) })
	return result, nil
}

func (s *MemoryService) TransitionIncident(ctx context.Context, id string, target IncidentStatus, input IncidentTransitionInput) (FoodSafetyIncident, error) {
	if err := checkContext(ctx); err != nil {
		return FoodSafetyIncident{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return FoodSafetyIncident{}, validationError(FieldError{"actor_id", "is required"})
	}
	if !target.Valid() {
		return FoodSafetyIncident{}, validationError(FieldError{"status", "is invalid"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[id]
	if !ok {
		return FoodSafetyIncident{}, ErrNotFound
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
	if target == IncidentClosed && len(incident.RecallIDs) > 0 && s.hasActiveRecallLocked(incident.RecallIDs) {
		return FoodSafetyIncident{}, fmt.Errorf("%w: all recalls must be completed or cancelled before closing", ErrInvalidState)
	}
	if err := s.validateEvidenceRefsLocked(EvidenceSubject{Type: EvidenceIncident, ID: id}, input.EvidenceIDs); err != nil {
		return FoodSafetyIncident{}, err
	}
	if target == IncidentContained {
		for _, batchID := range incident.BatchIDs {
			if err := s.canQuarantineBatchLocked(batchID); err != nil {
				return FoodSafetyIncident{}, err
			}
		}
	}
	if input.RegulatoryReport != nil {
		if input.RegulatoryReport.Reported && input.RegulatoryReport.ReportedAt == nil {
			return FoodSafetyIncident{}, validationError(FieldError{"regulatory_report.reported_at", "is required when reported is true"})
		}
		incident.RegulatoryReport = cloneRegulatoryReport(*input.RegulatoryReport)
	}
	now := s.currentTime()
	from := incident.Status
	if target == IncidentContained {
		incident.ContainmentAction = strings.TrimSpace(input.ContainmentAction)
		incident.ContainedAt = timePtr(now)
		for _, batchID := range incident.BatchIDs {
			if err := s.quarantineBatchLocked(batchID, input.ActorID, incident.ContainmentAction, now); err != nil {
				return FoodSafetyIncident{}, err
			}
		}
	}
	if target == IncidentInvestigating {
		incident.InvestigationSummary = strings.TrimSpace(input.InvestigationSummary)
	}
	if target == IncidentResolved {
		incident.ResolutionSummary = strings.TrimSpace(input.ResolutionSummary)
		incident.ResolvedAt = timePtr(now)
	}
	if target == IncidentClosed {
		incident.ClosedAt = timePtr(now)
	}
	incident.Status = target
	incident.EvidenceIDs = appendUnique(incident.EvidenceIDs, input.EvidenceIDs...)
	incident.UpdatedAt = now
	appendStatusChange(&incident.StatusHistory, string(from), string(target), input.ActorID, input.Reason, now)
	s.incidents[id] = incident
	return cloneIncident(incident), nil
}

func (s *MemoryService) CreateRecall(ctx context.Context, input CreateRecallInput) (Recall, error) {
	if err := checkContext(ctx); err != nil {
		return Recall{}, err
	}
	if len(input.EvidenceIDs) > 0 {
		return Recall{}, fmt.Errorf("%w: add recall evidence after creation", ErrConflict)
	}
	recall := Recall{
		ID:               newID("recall"),
		IncidentID:       strings.TrimSpace(input.IncidentID),
		MerchantID:       strings.TrimSpace(input.MerchantID),
		Scope:            input.Scope,
		Reason:           strings.TrimSpace(input.Reason),
		BatchIDs:         uniqueStrings(input.BatchIDs),
		OrderIDs:         uniqueStrings(input.OrderIDs),
		AffectedQuantity: input.AffectedQuantity,
		Status:           RecallDraft,
		CreatedAt:        s.currentTime(),
		UpdatedAt:        s.currentTime(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[recall.IncidentID]
	if !ok {
		return Recall{}, ErrNotFound
	}
	if recall.MerchantID == "" {
		recall.MerchantID = incident.MerchantID
	}
	if recall.MerchantID != incident.MerchantID {
		return Recall{}, ErrConflict
	}
	if err := ValidateRecall(recall); err != nil {
		return Recall{}, err
	}
	if incident.Status != IncidentConfirmed && incident.Status != IncidentRecalling && incident.Status != IncidentCompensating {
		return Recall{}, fmt.Errorf("%w: an incident must be confirmed before a recall is created", ErrInvalidState)
	}
	if err := s.validateRecallTargetsLocked(incident, recall); err != nil {
		return Recall{}, err
	}
	s.recalls[recall.ID] = cloneRecall(recall)
	incident.RecallIDs = appendUnique(incident.RecallIDs, recall.ID)
	incident.UpdatedAt = s.currentTime()
	s.incidents[incident.ID] = incident
	for _, batchID := range recall.BatchIDs {
		batch := s.batches[batchID]
		batch.RecallIDs = appendUnique(batch.RecallIDs, recall.ID)
		batch.UpdatedAt = s.currentTime()
		s.batches[batchID] = batch
	}
	return cloneRecall(recall), nil
}

func (s *MemoryService) GetRecall(ctx context.Context, id string) (Recall, error) {
	if err := checkContext(ctx); err != nil {
		return Recall{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	recall, ok := s.recalls[id]
	if !ok {
		return Recall{}, ErrNotFound
	}
	return cloneRecall(recall), nil
}

func (s *MemoryService) ListRecalls(ctx context.Context, merchantID string, status RecallStatus) ([]Recall, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Recall, 0)
	for _, recall := range s.recalls {
		if merchantID != "" && recall.MerchantID != merchantID {
			continue
		}
		if status != "" && recall.Status != status {
			continue
		}
		result = append(result, cloneRecall(recall))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *MemoryService) TransitionRecall(ctx context.Context, id string, target RecallStatus, input RecallTransitionInput) (Recall, error) {
	if err := checkContext(ctx); err != nil {
		return Recall{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return Recall{}, validationError(FieldError{"actor_id", "is required"})
	}
	if !target.Valid() {
		return Recall{}, validationError(FieldError{"status", "is invalid"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	recall, ok := s.recalls[id]
	if !ok {
		return Recall{}, ErrNotFound
	}
	if !CanTransitionRecall(recall.Status, target) {
		return Recall{}, invalidTransition(string(recall.Status), string(target))
	}
	incident, ok := s.incidents[recall.IncidentID]
	if !ok {
		return Recall{}, ErrNotFound
	}
	if err := s.validateEvidenceRefsLocked(EvidenceSubject{Type: EvidenceRecall, ID: id}, input.EvidenceIDs); err != nil {
		return Recall{}, err
	}
	now := s.currentTime()
	if target == RecallInitiated {
		if incident.Status != IncidentConfirmed && incident.Status != IncidentRecalling && incident.Status != IncidentCompensating {
			return Recall{}, fmt.Errorf("%w: incident is not ready to initiate recall", ErrInvalidState)
		}
		for _, batchID := range recall.BatchIDs {
			if err := s.canMarkBatchRecalledLocked(batchID); err != nil {
				return Recall{}, err
			}
		}
		for _, batchID := range recall.BatchIDs {
			s.markBatchRecalledLocked(batchID, input.ActorID, recall.Reason, now)
		}
		if incident.Status == IncidentConfirmed {
			appendStatusChange(&incident.StatusHistory, string(incident.Status), string(IncidentRecalling), input.ActorID, recall.Reason, now)
			incident.Status = IncidentRecalling
		}
		recall.InitiatedBy = input.ActorID
		notifiedAt := input.NotifiedAt
		if notifiedAt == nil {
			notifiedAt = timePtr(now)
		}
		recall.NotifiedAt = notifiedAt
		s.incidents[incident.ID] = incident
	}
	if target == RecallCompleted {
		if input.RecoveredQuantity < 0 || input.DisposedQuantity < 0 ||
			input.RecoveredQuantity+input.DisposedQuantity > recall.AffectedQuantity {
			return Recall{}, validationError(FieldError{"quantities", "recovered and disposed quantities exceed affected quantity"})
		}
		recall.RecoveredQuantity = input.RecoveredQuantity
		recall.DisposedQuantity = input.DisposedQuantity
		recall.CompletedAt = timePtr(now)
	}
	if target == RecallCancelled && strings.TrimSpace(input.Reason) == "" {
		return Recall{}, validationError(FieldError{"reason", "is required when cancelling a recall"})
	}
	from := recall.Status
	recall.Status = target
	recall.EvidenceIDs = appendUnique(recall.EvidenceIDs, input.EvidenceIDs...)
	recall.UpdatedAt = now
	appendStatusChange(&recall.StatusHistory, string(from), string(target), input.ActorID, input.Reason, now)
	s.recalls[id] = recall
	return cloneRecall(recall), nil
}

func (s *MemoryService) CreateDisposition(ctx context.Context, input CreateDispositionInput) (Disposition, error) {
	if err := checkContext(ctx); err != nil {
		return Disposition{}, err
	}
	if len(input.EvidenceIDs) > 0 {
		return Disposition{}, fmt.Errorf("%w: add disposition evidence after creation", ErrConflict)
	}
	disposition := Disposition{
		ID:         newID("disposition"),
		IncidentID: strings.TrimSpace(input.IncidentID),
		RecallID:   strings.TrimSpace(input.RecallID),
		BatchID:    strings.TrimSpace(input.BatchID),
		OrderID:    strings.TrimSpace(input.OrderID),
		Type:       input.Type,
		Quantity:   input.Quantity,
		Unit:       strings.TrimSpace(input.Unit),
		Status:     DispositionPlanned,
		ActionBy:   strings.TrimSpace(input.ActionBy),
		Notes:      strings.TrimSpace(input.Notes),
		CreatedAt:  s.currentTime(),
		UpdatedAt:  s.currentTime(),
	}
	if err := ValidateDisposition(disposition); err != nil {
		return Disposition{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	incident, ok := s.incidents[disposition.IncidentID]
	if !ok {
		return Disposition{}, ErrNotFound
	}
	if incident.Status == IncidentClosed || incident.Status == IncidentDismissed {
		return Disposition{}, fmt.Errorf("%w: dispositions cannot be created after incident closure", ErrInvalidState)
	}
	if disposition.RecallID != "" {
		recall, ok := s.recalls[disposition.RecallID]
		if !ok {
			return Disposition{}, ErrNotFound
		}
		if recall.IncidentID != incident.ID {
			return Disposition{}, ErrConflict
		}
	}
	if err := s.validateDispositionTargetLocked(incident, disposition); err != nil {
		return Disposition{}, err
	}
	s.dispositions[disposition.ID] = cloneDisposition(disposition)
	incident.DispositionIDs = appendUnique(incident.DispositionIDs, disposition.ID)
	incident.UpdatedAt = s.currentTime()
	s.incidents[incident.ID] = incident
	if disposition.RecallID != "" {
		recall := s.recalls[disposition.RecallID]
		recall.DispositionIDs = appendUnique(recall.DispositionIDs, disposition.ID)
		recall.UpdatedAt = s.currentTime()
		s.recalls[recall.ID] = recall
	}
	return cloneDisposition(disposition), nil
}

func (s *MemoryService) GetDisposition(ctx context.Context, id string) (Disposition, error) {
	if err := checkContext(ctx); err != nil {
		return Disposition{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	disposition, ok := s.dispositions[id]
	if !ok {
		return Disposition{}, ErrNotFound
	}
	return cloneDisposition(disposition), nil
}

func (s *MemoryService) TransitionDisposition(ctx context.Context, id string, target DispositionStatus, input TransitionInput) (Disposition, error) {
	if err := checkContext(ctx); err != nil {
		return Disposition{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return Disposition{}, validationError(FieldError{"actor_id", "is required"})
	}
	if !target.Valid() {
		return Disposition{}, validationError(FieldError{"status", "is invalid"})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	disposition, ok := s.dispositions[id]
	if !ok {
		return Disposition{}, ErrNotFound
	}
	if !CanTransitionDisposition(disposition.Status, target) {
		return Disposition{}, invalidTransition(string(disposition.Status), string(target))
	}
	if target == DispositionCompleted &&
		(disposition.Type == DispositionDestroy || disposition.Type == DispositionReturnToSupplier) &&
		len(input.EvidenceIDs) == 0 {
		return Disposition{}, fmt.Errorf("%w: completion evidence is required for physical disposition", ErrInvalidState)
	}
	if target != DispositionCompleted && strings.TrimSpace(input.Reason) == "" {
		return Disposition{}, validationError(FieldError{"reason", "is required when disposition does not complete"})
	}
	if err := s.validateEvidenceRefsLocked(EvidenceSubject{Type: EvidenceDisposition, ID: id}, input.EvidenceIDs); err != nil {
		return Disposition{}, err
	}
	if disposition.RecallID != "" && target == DispositionCompleted && disposition.Type == DispositionDestroy {
		recall := s.recalls[disposition.RecallID]
		if recall.DisposedQuantity+disposition.Quantity > recall.AffectedQuantity {
			return Disposition{}, fmt.Errorf("%w: disposed quantity exceeds recall quantity", ErrConflict)
		}
	}
	now := s.currentTime()
	from := disposition.Status
	disposition.Status = target
	disposition.EvidenceIDs = appendUnique(disposition.EvidenceIDs, input.EvidenceIDs...)
	if target == DispositionCompleted {
		disposition.CompletedAt = timePtr(now)
	}
	disposition.UpdatedAt = now
	appendStatusChange(&disposition.StatusHistory, string(from), string(target), input.ActorID, input.Reason, now)
	s.dispositions[id] = disposition
	if disposition.RecallID != "" && target == DispositionCompleted {
		recall := s.recalls[disposition.RecallID]
		if disposition.Type == DispositionDestroy {
			recall.DisposedQuantity += disposition.Quantity
			recall.UpdatedAt = now
			s.recalls[recall.ID] = recall
		}
	}
	return cloneDisposition(disposition), nil
}

func (s *MemoryService) AddEvidence(ctx context.Context, subject EvidenceSubject, input AddEvidenceInput) (Evidence, error) {
	if err := checkContext(ctx); err != nil {
		return Evidence{}, err
	}
	subject.Type = EvidenceSubjectType(strings.TrimSpace(string(subject.Type)))
	subject.ID = strings.TrimSpace(subject.ID)
	evidence := Evidence{
		ID:          newID("evidence"),
		Subject:     subject,
		Sequence:    1,
		Kind:        input.Kind,
		URI:         strings.TrimSpace(input.URI),
		SHA256:      strings.ToLower(strings.TrimSpace(input.SHA256)),
		MimeType:    strings.TrimSpace(input.MimeType),
		CapturedBy:  strings.TrimSpace(input.CapturedBy),
		CapturedAt:  input.CapturedAt,
		Description: strings.TrimSpace(input.Description),
		Metadata:    cloneStringMap(input.Metadata),
		CreatedAt:   s.currentTime(),
	}
	if evidence.CapturedAt.IsZero() {
		evidence.CapturedAt = evidence.CreatedAt
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.subjectExistsLocked(subject) {
		return Evidence{}, ErrNotFound
	}
	key := subjectKey(subject)
	chain := s.evidenceChains[key]
	evidence.Sequence = uint64(len(chain) + 1)
	if len(chain) > 0 {
		evidence.PreviousHash = s.evidence[chain[len(chain)-1]].Hash
	}
	evidence.Hash = evidenceHash(evidence)
	if err := ValidateEvidence(evidence); err != nil {
		return Evidence{}, err
	}
	s.evidence[evidence.ID] = cloneEvidence(evidence)
	s.evidenceChains[key] = append(chain, evidence.ID)
	if err := s.appendEvidenceLinkLocked(subject, evidence.ID); err != nil {
		return Evidence{}, err
	}
	return cloneEvidence(evidence), nil
}

func (s *MemoryService) ListEvidence(ctx context.Context, subject EvidenceSubject) ([]Evidence, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.subjectExistsLocked(subject) {
		return nil, ErrNotFound
	}
	ids := s.evidenceChains[subjectKey(subject)]
	result := make([]Evidence, 0, len(ids))
	for _, id := range ids {
		result = append(result, cloneEvidence(s.evidence[id]))
	}
	return result, nil
}

func (s *MemoryService) VerifyEvidenceChain(ctx context.Context, subject EvidenceSubject) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.subjectExistsLocked(subject) {
		return ErrNotFound
	}
	ids := s.evidenceChains[subjectKey(subject)]
	var previous string
	for index, id := range ids {
		evidence, ok := s.evidence[id]
		if !ok {
			return fmt.Errorf("%w: evidence %s is missing", ErrConflict, id)
		}
		if evidence.Sequence != uint64(index+1) || evidence.PreviousHash != previous ||
			evidence.Hash != evidenceHash(evidence) {
			return fmt.Errorf("%w: evidence chain is invalid at sequence %d", ErrConflict, index+1)
		}
		previous = evidence.Hash
	}
	return nil
}

func (s *MemoryService) approvedQualificationLocked(merchantID string, now time.Time) (MerchantQualification, bool) {
	var selected MerchantQualification
	found := false
	for _, qualification := range s.qualifications {
		if qualification.MerchantID != merchantID || qualification.Status != QualificationApproved ||
			!qualification.BusinessLicenseExpiresAt.After(now) || !qualification.FoodPermitExpiresAt.After(now) {
			continue
		}
		if !found || qualification.UpdatedAt.After(selected.UpdatedAt) {
			selected = qualification
			found = true
		}
	}
	return selected, found
}

func (s *MemoryService) validateIncidentBatchesLocked(merchantID string, batchIDs []string) error {
	for _, batchID := range batchIDs {
		batch, ok := s.batches[batchID]
		if !ok {
			return ErrNotFound
		}
		if batch.MerchantID != merchantID {
			return ErrConflict
		}
	}
	return nil
}

func (s *MemoryService) validateRecallTargetsLocked(incident FoodSafetyIncident, recall Recall) error {
	if recall.Scope == RecallBatch && len(recall.BatchIDs) == 0 {
		return validationError(FieldError{"batch_ids", "is required for BATCH recall"})
	}
	if recall.Scope == RecallOrder && len(recall.OrderIDs) == 0 {
		return validationError(FieldError{"order_ids", "is required for ORDER recall"})
	}
	for _, batchID := range recall.BatchIDs {
		if !containsString(incident.BatchIDs, batchID) {
			return fmt.Errorf("%w: batch %s is outside incident scope", ErrConflict, batchID)
		}
		if batch := s.batches[batchID]; batch.MerchantID != recall.MerchantID {
			return ErrConflict
		}
	}
	for _, orderID := range recall.OrderIDs {
		if !containsString(incident.OrderIDs, orderID) {
			return fmt.Errorf("%w: order %s is outside incident scope", ErrConflict, orderID)
		}
	}
	return nil
}

func (s *MemoryService) validateDispositionTargetLocked(incident FoodSafetyIncident, disposition Disposition) error {
	if disposition.BatchID != "" {
		batch, ok := s.batches[disposition.BatchID]
		if !ok {
			return ErrNotFound
		}
		if !containsString(incident.BatchIDs, disposition.BatchID) || batch.MerchantID != incident.MerchantID {
			return ErrConflict
		}
	}
	if disposition.OrderID != "" && !containsString(incident.OrderIDs, disposition.OrderID) {
		return ErrConflict
	}
	if disposition.RecallID != "" {
		recall := s.recalls[disposition.RecallID]
		if disposition.BatchID != "" && !containsString(recall.BatchIDs, disposition.BatchID) {
			return ErrConflict
		}
		if disposition.OrderID != "" && !containsString(recall.OrderIDs, disposition.OrderID) {
			return ErrConflict
		}
	}
	return nil
}

func (s *MemoryService) hasActiveRecallLocked(ids []string) bool {
	for _, id := range ids {
		recall, ok := s.recalls[id]
		if ok && recall.Status != RecallCompleted && recall.Status != RecallCancelled {
			return true
		}
	}
	return false
}

func (s *MemoryService) quarantineBatchLocked(id, actorID, reason string, now time.Time) error {
	batch, ok := s.batches[id]
	if !ok {
		return ErrNotFound
	}
	if err := s.canQuarantineBatchLocked(id); err != nil {
		return err
	}
	if batch.Status == BatchQuarantined || batch.Status == BatchRecalled || batch.Status == BatchDisposed || batch.Status == BatchCancelled {
		return nil
	}
	appendStatusChange(&batch.StatusHistory, string(batch.Status), string(BatchQuarantined), actorID, reason, now)
	batch.Status = BatchQuarantined
	batch.UpdatedAt = now
	s.batches[id] = batch
	return nil
}

func (s *MemoryService) canQuarantineBatchLocked(id string) error {
	batch, ok := s.batches[id]
	if !ok {
		return ErrNotFound
	}
	if batch.Status == BatchQuarantined || batch.Status == BatchRecalled ||
		batch.Status == BatchDisposed || batch.Status == BatchCancelled {
		return nil
	}
	if !CanTransitionBatch(batch.Status, BatchQuarantined) {
		return fmt.Errorf("%w: cannot quarantine batch %s from %s", ErrInvalidTransition, id, batch.Status)
	}
	return nil
}

func (s *MemoryService) canMarkBatchRecalledLocked(id string) error {
	batch, ok := s.batches[id]
	if !ok {
		return ErrNotFound
	}
	if batch.Status == BatchRecalled {
		return nil
	}
	if batch.Status == BatchDisposed || batch.Status == BatchCancelled || batch.QuantityProduced <= 0 {
		return fmt.Errorf("%w: batch %s cannot be recalled from %s", ErrInvalidState, id, batch.Status)
	}
	if !CanTransitionBatch(batch.Status, BatchRecalled) {
		return invalidTransition(string(batch.Status), string(BatchRecalled))
	}
	return nil
}

func (s *MemoryService) markBatchRecalledLocked(id, actorID, reason string, now time.Time) {
	batch := s.batches[id]
	if batch.Status == BatchRecalled {
		return
	}
	appendStatusChange(&batch.StatusHistory, string(batch.Status), string(BatchRecalled), actorID, reason, now)
	batch.Status = BatchRecalled
	batch.UpdatedAt = now
	s.batches[id] = batch
}

func (s *MemoryService) validateEvidenceRefsLocked(subject EvidenceSubject, ids []string) error {
	for _, id := range uniqueStrings(ids) {
		evidence, ok := s.evidence[id]
		if !ok {
			return ErrNotFound
		}
		if evidence.Subject != subject {
			return fmt.Errorf("%w: evidence %s belongs to another subject", ErrConflict, id)
		}
	}
	return nil
}

func (s *MemoryService) subjectExistsLocked(subject EvidenceSubject) bool {
	switch subject.Type {
	case EvidenceQualification:
		_, ok := s.qualifications[subject.ID]
		return ok
	case EvidenceBatch:
		_, ok := s.batches[subject.ID]
		return ok
	case EvidenceIncident:
		_, ok := s.incidents[subject.ID]
		return ok
	case EvidenceRecall:
		_, ok := s.recalls[subject.ID]
		return ok
	case EvidenceDisposition:
		_, ok := s.dispositions[subject.ID]
		return ok
	default:
		return false
	}
}

func (s *MemoryService) appendEvidenceLinkLocked(subject EvidenceSubject, evidenceID string) error {
	switch subject.Type {
	case EvidenceQualification:
		qualification := s.qualifications[subject.ID]
		qualification.EvidenceIDs = appendUnique(qualification.EvidenceIDs, evidenceID)
		qualification.UpdatedAt = s.currentTime()
		s.qualifications[subject.ID] = qualification
	case EvidenceBatch:
		batch := s.batches[subject.ID]
		batch.EvidenceIDs = appendUnique(batch.EvidenceIDs, evidenceID)
		batch.UpdatedAt = s.currentTime()
		s.batches[subject.ID] = batch
	case EvidenceIncident:
		incident := s.incidents[subject.ID]
		incident.EvidenceIDs = appendUnique(incident.EvidenceIDs, evidenceID)
		incident.UpdatedAt = s.currentTime()
		s.incidents[subject.ID] = incident
	case EvidenceRecall:
		recall := s.recalls[subject.ID]
		recall.EvidenceIDs = appendUnique(recall.EvidenceIDs, evidenceID)
		recall.UpdatedAt = s.currentTime()
		s.recalls[subject.ID] = recall
	case EvidenceDisposition:
		disposition := s.dispositions[subject.ID]
		disposition.EvidenceIDs = appendUnique(disposition.EvidenceIDs, evidenceID)
		disposition.UpdatedAt = s.currentTime()
		s.dispositions[subject.ID] = disposition
	default:
		return validationError(FieldError{"subject.type", "is invalid"})
	}
	return nil
}

func newID(prefix string) string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(buffer)
}

func timePtr(value time.Time) *time.Time {
	copy := value
	return &copy
}

func appendStatusChange(history *[]StatusChange, from, to, actorID, reason string, at time.Time) {
	*history = append(*history, StatusChange{
		From:    from,
		To:      to,
		ActorID: actorID,
		Reason:  strings.TrimSpace(reason),
		At:      at,
	})
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func appendUnique(values []string, additions ...string) []string {
	return uniqueStrings(append(append([]string(nil), values...), additions...))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func cloneDocuments(values []QualificationDocument) []QualificationDocument {
	if values == nil {
		return nil
	}
	result := make([]QualificationDocument, len(values))
	for i, value := range values {
		result[i] = value
		result[i].ExpiresAt = cloneTimePtr(value.ExpiresAt)
	}
	return result
}

func cloneReviews(values []QualificationReview) []QualificationReview {
	if values == nil {
		return nil
	}
	result := make([]QualificationReview, len(values))
	for i, value := range values {
		result[i] = value
		result[i].EvidenceIDs = cloneStrings(value.EvidenceIDs)
	}
	return result
}

func cloneInspection(value *SiteInspection) *SiteInspection {
	if value == nil {
		return nil
	}
	result := *value
	result.EvidenceIDs = cloneStrings(value.EvidenceIDs)
	return &result
}

func cloneQualification(value MerchantQualification) MerchantQualification {
	value.BusinessScope = cloneStrings(value.BusinessScope)
	value.Documents = cloneDocuments(value.Documents)
	value.SiteInspection = cloneInspection(value.SiteInspection)
	value.ReviewHistory = cloneReviews(value.ReviewHistory)
	value.EvidenceIDs = cloneStrings(value.EvidenceIDs)
	value.SubmittedAt = cloneTimePtr(value.SubmittedAt)
	value.ReviewedAt = cloneTimePtr(value.ReviewedAt)
	value.StatusHistory = cloneStatusHistory(value.StatusHistory)
	return value
}

func cloneProductSpec(value ProductSpec) ProductSpec {
	value.Ingredients = cloneStrings(value.Ingredients)
	value.Allergens = cloneStrings(value.Allergens)
	return value
}

func cloneIngredientLots(values []IngredientLot) []IngredientLot {
	if values == nil {
		return nil
	}
	return append([]IngredientLot(nil), values...)
}

func cloneOrderAssociations(values []OrderAssociation) []OrderAssociation {
	if values == nil {
		return nil
	}
	return append([]OrderAssociation(nil), values...)
}

func cloneStatusHistory(values []StatusChange) []StatusChange {
	if values == nil {
		return nil
	}
	return append([]StatusChange(nil), values...)
}

func cloneBatch(value FoodBatch) FoodBatch {
	value.Specification = cloneProductSpec(value.Specification)
	value.IngredientLots = cloneIngredientLots(value.IngredientLots)
	value.Orders = cloneOrderAssociations(value.Orders)
	value.RecallIDs = cloneStrings(value.RecallIDs)
	value.EvidenceIDs = cloneStrings(value.EvidenceIDs)
	value.StatusHistory = cloneStatusHistory(value.StatusHistory)
	value.ProducedAt = cloneTimePtr(value.ProducedAt)
	value.ExpiresAt = cloneTimePtr(value.ExpiresAt)
	return value
}

func cloneRegulatoryReport(value RegulatoryReport) RegulatoryReport {
	value.ReportedAt = cloneTimePtr(value.ReportedAt)
	return value
}

func cloneIncident(value FoodSafetyIncident) FoodSafetyIncident {
	value.BatchIDs = cloneStrings(value.BatchIDs)
	value.OrderIDs = cloneStrings(value.OrderIDs)
	value.RegulatoryReport = cloneRegulatoryReport(value.RegulatoryReport)
	value.RecallIDs = cloneStrings(value.RecallIDs)
	value.DispositionIDs = cloneStrings(value.DispositionIDs)
	value.EvidenceIDs = cloneStrings(value.EvidenceIDs)
	value.StatusHistory = cloneStatusHistory(value.StatusHistory)
	value.ContainedAt = cloneTimePtr(value.ContainedAt)
	value.ResolvedAt = cloneTimePtr(value.ResolvedAt)
	value.ClosedAt = cloneTimePtr(value.ClosedAt)
	return value
}

func cloneRecall(value Recall) Recall {
	value.BatchIDs = cloneStrings(value.BatchIDs)
	value.OrderIDs = cloneStrings(value.OrderIDs)
	value.EvidenceIDs = cloneStrings(value.EvidenceIDs)
	value.DispositionIDs = cloneStrings(value.DispositionIDs)
	value.StatusHistory = cloneStatusHistory(value.StatusHistory)
	value.NotifiedAt = cloneTimePtr(value.NotifiedAt)
	value.CompletedAt = cloneTimePtr(value.CompletedAt)
	return value
}

func cloneDisposition(value Disposition) Disposition {
	value.EvidenceIDs = cloneStrings(value.EvidenceIDs)
	value.StatusHistory = cloneStatusHistory(value.StatusHistory)
	value.CompletedAt = cloneTimePtr(value.CompletedAt)
	return value
}

func cloneEvidence(value Evidence) Evidence {
	value.Metadata = cloneStringMap(value.Metadata)
	return value
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func subjectKey(subject EvidenceSubject) string {
	return string(subject.Type) + ":" + subject.ID
}
