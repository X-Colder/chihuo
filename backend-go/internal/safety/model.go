package safety

import "time"

type QualificationStatus string

const (
	QualificationDraft         QualificationStatus = "DRAFT"
	QualificationPendingReview QualificationStatus = "PENDING_REVIEW"
	QualificationApproved      QualificationStatus = "APPROVED"
	QualificationRejected      QualificationStatus = "REJECTED"
	QualificationSuspended     QualificationStatus = "SUSPENDED"
	QualificationExpired       QualificationStatus = "EXPIRED"
)

func (s QualificationStatus) Valid() bool {
	switch s {
	case QualificationDraft, QualificationPendingReview, QualificationApproved,
		QualificationRejected, QualificationSuspended, QualificationExpired:
		return true
	default:
		return false
	}
}

type QualificationDocumentKind string

const (
	DocumentBusinessLicense QualificationDocumentKind = "BUSINESS_LICENSE"
	DocumentFoodPermit      QualificationDocumentKind = "FOOD_PERMIT"
	DocumentSitePhoto       QualificationDocumentKind = "SITE_PHOTO"
	DocumentOther           QualificationDocumentKind = "OTHER"
)

func (k QualificationDocumentKind) Valid() bool {
	switch k {
	case DocumentBusinessLicense, DocumentFoodPermit, DocumentSitePhoto, DocumentOther:
		return true
	default:
		return false
	}
}

type QualificationDocument struct {
	ID         string                    `json:"id"`
	Kind       QualificationDocumentKind `json:"kind"`
	URI        string                    `json:"uri"`
	SHA256     string                    `json:"sha256"`
	IssuedAt   time.Time                 `json:"issued_at"`
	ExpiresAt  *time.Time                `json:"expires_at,omitempty"`
	UploadedBy string                    `json:"uploaded_by"`
	UploadedAt time.Time                 `json:"uploaded_at"`
}

type QualificationReview struct {
	ID              string              `json:"id"`
	QualificationID string              `json:"qualification_id"`
	ReviewerID      string              `json:"reviewer_id"`
	FromStatus      QualificationStatus `json:"from_status"`
	ToStatus        QualificationStatus `json:"to_status"`
	Reason          string              `json:"reason,omitempty"`
	EvidenceIDs     []string            `json:"evidence_ids,omitempty"`
	ReviewedAt      time.Time           `json:"reviewed_at"`
}

type SiteInspectionResult string

const (
	SiteInspectionPending SiteInspectionResult = "PENDING"
	SiteInspectionPassed  SiteInspectionResult = "PASSED"
	SiteInspectionFailed  SiteInspectionResult = "FAILED"
)

func (r SiteInspectionResult) Valid() bool {
	switch r {
	case SiteInspectionPending, SiteInspectionPassed, SiteInspectionFailed:
		return true
	default:
		return false
	}
}

type SiteInspection struct {
	ID              string               `json:"id"`
	QualificationID string               `json:"qualification_id"`
	InspectorID     string               `json:"inspector_id"`
	Result          SiteInspectionResult `json:"result"`
	Notes           string               `json:"notes,omitempty"`
	EvidenceIDs     []string             `json:"evidence_ids,omitempty"`
	InspectedAt     time.Time            `json:"inspected_at"`
}

type MerchantQualification struct {
	ID                       string                  `json:"id"`
	MerchantID               string                  `json:"merchant_id"`
	LegalEntityName          string                  `json:"legal_entity_name"`
	StoreName                string                  `json:"store_name"`
	BusinessLicenseNumber    string                  `json:"business_license_number"`
	FoodPermitNumber         string                  `json:"food_permit_number"`
	RegisteredAddress        string                  `json:"registered_address"`
	OperatingAddress         string                  `json:"operating_address"`
	BusinessLicenseIssuedAt  time.Time               `json:"business_license_issued_at"`
	BusinessLicenseExpiresAt time.Time               `json:"business_license_expires_at"`
	FoodPermitIssuedAt       time.Time               `json:"food_permit_issued_at"`
	FoodPermitExpiresAt      time.Time               `json:"food_permit_expires_at"`
	BusinessScope            []string                `json:"business_scope"`
	Documents                []QualificationDocument `json:"documents"`
	Status                   QualificationStatus     `json:"status"`
	SiteInspection           *SiteInspection         `json:"site_inspection,omitempty"`
	ReviewHistory            []QualificationReview   `json:"review_history,omitempty"`
	StatusHistory            []StatusChange          `json:"status_history,omitempty"`
	EvidenceIDs              []string                `json:"evidence_ids,omitempty"`
	SubmittedAt              *time.Time              `json:"submitted_at,omitempty"`
	ReviewedAt               *time.Time              `json:"reviewed_at,omitempty"`
	CreatedAt                time.Time               `json:"created_at"`
	UpdatedAt                time.Time               `json:"updated_at"`
}

type ProductSpec struct {
	WeightGrams         int      `json:"weight_grams"`
	Ingredients         []string `json:"ingredients"`
	Allergens           []string `json:"allergens"`
	OilLevel            string   `json:"oil_level"`
	SaltLevel           string   `json:"salt_level"`
	StorageInstructions string   `json:"storage_instructions"`
}

type IngredientLot struct {
	Ingredient    string    `json:"ingredient"`
	Supplier      string    `json:"supplier"`
	LotNumber     string    `json:"lot_number"`
	ReceivedAt    time.Time `json:"received_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	QuantityGrams int       `json:"quantity_grams"`
}

type OrderAssociation struct {
	OrderID  string    `json:"order_id"`
	Quantity int       `json:"quantity"`
	LinkedBy string    `json:"linked_by"`
	LinkedAt time.Time `json:"linked_at"`
}

type FoodBatchStatus string

const (
	BatchDraft           FoodBatchStatus = "DRAFT"
	BatchScheduled       FoodBatchStatus = "SCHEDULED"
	BatchProducing       FoodBatchStatus = "PRODUCING"
	BatchPacked          FoodBatchStatus = "PACKED"
	BatchReadyForHandoff FoodBatchStatus = "READY_FOR_HANDOFF"
	BatchInDelivery      FoodBatchStatus = "IN_DELIVERY"
	BatchCompleted       FoodBatchStatus = "COMPLETED"
	BatchQuarantined     FoodBatchStatus = "QUARANTINED"
	BatchRecalled        FoodBatchStatus = "RECALLED"
	BatchDisposed        FoodBatchStatus = "DISPOSED"
	BatchCancelled       FoodBatchStatus = "CANCELLED"
)

func (s FoodBatchStatus) Valid() bool {
	switch s {
	case BatchDraft, BatchScheduled, BatchProducing, BatchPacked,
		BatchReadyForHandoff, BatchInDelivery, BatchCompleted,
		BatchQuarantined, BatchRecalled, BatchDisposed, BatchCancelled:
		return true
	default:
		return false
	}
}

type StatusChange struct {
	From    string    `json:"from"`
	To      string    `json:"to"`
	ActorID string    `json:"actor_id"`
	Reason  string    `json:"reason,omitempty"`
	At      time.Time `json:"at"`
}

type FoodBatch struct {
	ID                string             `json:"id"`
	MerchantID        string             `json:"merchant_id"`
	ProductID         string             `json:"product_id"`
	CampaignID        string             `json:"campaign_id,omitempty"`
	ProductionDate    time.Time          `json:"production_date"`
	ProducedAt        *time.Time         `json:"produced_at,omitempty"`
	ExpiresAt         *time.Time         `json:"expires_at,omitempty"`
	ShelfLifeMinutes  int                `json:"shelf_life_minutes"`
	StorageCondition  string             `json:"storage_condition"`
	QuantityPlanned   int                `json:"quantity_planned"`
	QuantityProduced  int                `json:"quantity_produced"`
	QuantityRemaining int                `json:"quantity_remaining"`
	UnitWeightGrams   int                `json:"unit_weight_grams"`
	Specification     ProductSpec        `json:"specification"`
	IngredientLots    []IngredientLot    `json:"ingredient_lots"`
	Orders            []OrderAssociation `json:"orders,omitempty"`
	RecallIDs         []string           `json:"recall_ids,omitempty"`
	EvidenceIDs       []string           `json:"evidence_ids,omitempty"`
	Status            FoodBatchStatus    `json:"status"`
	StatusHistory     []StatusChange     `json:"status_history,omitempty"`
	CreatedBy         string             `json:"created_by"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type IncidentSeverity string

const (
	SeverityLow      IncidentSeverity = "LOW"
	SeverityMedium   IncidentSeverity = "MEDIUM"
	SeverityHigh     IncidentSeverity = "HIGH"
	SeverityCritical IncidentSeverity = "CRITICAL"
)

func (s IncidentSeverity) Valid() bool {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	default:
		return false
	}
}

type IncidentStatus string

const (
	IncidentReported      IncidentStatus = "REPORTED"
	IncidentContained     IncidentStatus = "CONTAINED"
	IncidentInvestigating IncidentStatus = "INVESTIGATING"
	IncidentConfirmed     IncidentStatus = "CONFIRMED"
	IncidentRecalling     IncidentStatus = "RECALLING"
	IncidentCompensating  IncidentStatus = "COMPENSATING"
	IncidentResolved      IncidentStatus = "RESOLVED"
	IncidentClosed        IncidentStatus = "CLOSED"
	IncidentDismissed     IncidentStatus = "DISMISSED"
)

func (s IncidentStatus) Valid() bool {
	switch s {
	case IncidentReported, IncidentContained, IncidentInvestigating,
		IncidentConfirmed, IncidentRecalling, IncidentCompensating,
		IncidentResolved, IncidentClosed, IncidentDismissed:
		return true
	default:
		return false
	}
}

type RegulatoryReport struct {
	Reported   bool       `json:"reported"`
	Authority  string     `json:"authority,omitempty"`
	Reference  string     `json:"reference,omitempty"`
	ReportedAt *time.Time `json:"reported_at,omitempty"`
	Notes      string     `json:"notes,omitempty"`
}

type FoodSafetyIncident struct {
	ID                   string           `json:"id"`
	MerchantID           string           `json:"merchant_id"`
	ReportedBy           string           `json:"reported_by"`
	Category             string           `json:"category"`
	Severity             IncidentSeverity `json:"severity"`
	Title                string           `json:"title"`
	Description          string           `json:"description"`
	BatchIDs             []string         `json:"batch_ids,omitempty"`
	OrderIDs             []string         `json:"order_ids,omitempty"`
	Status               IncidentStatus   `json:"status"`
	ContainmentAction    string           `json:"containment_action,omitempty"`
	InvestigationSummary string           `json:"investigation_summary,omitempty"`
	ResolutionSummary    string           `json:"resolution_summary,omitempty"`
	RegulatoryReport     RegulatoryReport `json:"regulatory_report"`
	RecallIDs            []string         `json:"recall_ids,omitempty"`
	DispositionIDs       []string         `json:"disposition_ids,omitempty"`
	EvidenceIDs          []string         `json:"evidence_ids,omitempty"`
	StatusHistory        []StatusChange   `json:"status_history,omitempty"`
	ReportedAt           time.Time        `json:"reported_at"`
	ContainedAt          *time.Time       `json:"contained_at,omitempty"`
	ResolvedAt           *time.Time       `json:"resolved_at,omitempty"`
	ClosedAt             *time.Time       `json:"closed_at,omitempty"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

type RecallScope string

const (
	RecallBatch  RecallScope = "BATCH"
	RecallOrder  RecallScope = "ORDER"
	RecallMarket RecallScope = "MARKET"
)

func (s RecallScope) Valid() bool {
	switch s {
	case RecallBatch, RecallOrder, RecallMarket:
		return true
	default:
		return false
	}
}

type RecallStatus string

const (
	RecallDraft      RecallStatus = "DRAFT"
	RecallInitiated  RecallStatus = "INITIATED"
	RecallInProgress RecallStatus = "IN_PROGRESS"
	RecallCompleted  RecallStatus = "COMPLETED"
	RecallCancelled  RecallStatus = "CANCELLED"
)

func (s RecallStatus) Valid() bool {
	switch s {
	case RecallDraft, RecallInitiated, RecallInProgress, RecallCompleted, RecallCancelled:
		return true
	default:
		return false
	}
}

type Recall struct {
	ID                string         `json:"id"`
	IncidentID        string         `json:"incident_id"`
	MerchantID        string         `json:"merchant_id"`
	Scope             RecallScope    `json:"scope"`
	Reason            string         `json:"reason"`
	BatchIDs          []string       `json:"batch_ids,omitempty"`
	OrderIDs          []string       `json:"order_ids,omitempty"`
	AffectedQuantity  int            `json:"affected_quantity"`
	RecoveredQuantity int            `json:"recovered_quantity"`
	DisposedQuantity  int            `json:"disposed_quantity"`
	Status            RecallStatus   `json:"status"`
	EvidenceIDs       []string       `json:"evidence_ids,omitempty"`
	DispositionIDs    []string       `json:"disposition_ids,omitempty"`
	StatusHistory     []StatusChange `json:"status_history,omitempty"`
	InitiatedBy       string         `json:"initiated_by,omitempty"`
	NotifiedAt        *time.Time     `json:"notified_at,omitempty"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type DispositionType string

const (
	DispositionRefund           DispositionType = "REFUND"
	DispositionReplacement      DispositionType = "REPLACEMENT"
	DispositionDestroy          DispositionType = "DESTROY"
	DispositionReturnToSupplier DispositionType = "RETURN_TO_SUPPLIER"
	DispositionRelease          DispositionType = "RELEASE"
	DispositionOther            DispositionType = "OTHER"
)

func (t DispositionType) Valid() bool {
	switch t {
	case DispositionRefund, DispositionReplacement, DispositionDestroy,
		DispositionReturnToSupplier, DispositionRelease, DispositionOther:
		return true
	default:
		return false
	}
}

type DispositionStatus string

const (
	DispositionPlanned   DispositionStatus = "PLANNED"
	DispositionCompleted DispositionStatus = "COMPLETED"
	DispositionFailed    DispositionStatus = "FAILED"
	DispositionCancelled DispositionStatus = "CANCELLED"
)

func (s DispositionStatus) Valid() bool {
	switch s {
	case DispositionPlanned, DispositionCompleted, DispositionFailed, DispositionCancelled:
		return true
	default:
		return false
	}
}

type Disposition struct {
	ID            string            `json:"id"`
	IncidentID    string            `json:"incident_id"`
	RecallID      string            `json:"recall_id,omitempty"`
	BatchID       string            `json:"batch_id,omitempty"`
	OrderID       string            `json:"order_id,omitempty"`
	Type          DispositionType   `json:"type"`
	Quantity      int               `json:"quantity"`
	Unit          string            `json:"unit"`
	Status        DispositionStatus `json:"status"`
	ActionBy      string            `json:"action_by"`
	Notes         string            `json:"notes,omitempty"`
	EvidenceIDs   []string          `json:"evidence_ids,omitempty"`
	StatusHistory []StatusChange    `json:"status_history,omitempty"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type EvidenceSubjectType string

const (
	EvidenceQualification EvidenceSubjectType = "QUALIFICATION"
	EvidenceBatch         EvidenceSubjectType = "BATCH"
	EvidenceIncident      EvidenceSubjectType = "INCIDENT"
	EvidenceRecall        EvidenceSubjectType = "RECALL"
	EvidenceDisposition   EvidenceSubjectType = "DISPOSITION"
)

func (t EvidenceSubjectType) Valid() bool {
	switch t {
	case EvidenceQualification, EvidenceBatch, EvidenceIncident, EvidenceRecall, EvidenceDisposition:
		return true
	default:
		return false
	}
}

type EvidenceSubject struct {
	Type EvidenceSubjectType `json:"type"`
	ID   string              `json:"id"`
}

type EvidenceKind string

const (
	EvidenceDocument  EvidenceKind = "DOCUMENT"
	EvidencePhoto     EvidenceKind = "PHOTO"
	EvidenceVideo     EvidenceKind = "VIDEO"
	EvidenceRecord    EvidenceKind = "RECORD"
	EvidenceLabReport EvidenceKind = "LAB_REPORT"
	EvidenceMessage   EvidenceKind = "MESSAGE"
	EvidenceOther     EvidenceKind = "OTHER"
)

func (k EvidenceKind) Valid() bool {
	switch k {
	case EvidenceDocument, EvidencePhoto, EvidenceVideo, EvidenceRecord,
		EvidenceLabReport, EvidenceMessage, EvidenceOther:
		return true
	default:
		return false
	}
}

type Evidence struct {
	ID           string            `json:"id"`
	Subject      EvidenceSubject   `json:"subject"`
	Sequence     uint64            `json:"sequence"`
	PreviousHash string            `json:"previous_hash,omitempty"`
	Hash         string            `json:"hash"`
	Kind         EvidenceKind      `json:"kind"`
	URI          string            `json:"uri"`
	SHA256       string            `json:"sha256"`
	MimeType     string            `json:"mime_type,omitempty"`
	CapturedBy   string            `json:"captured_by"`
	CapturedAt   time.Time         `json:"captured_at"`
	Description  string            `json:"description,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}
