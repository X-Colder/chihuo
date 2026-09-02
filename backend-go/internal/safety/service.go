package safety

import (
	"context"
	"time"
)

type CreateQualificationInput struct {
	MerchantID               string
	CreatedBy                string
	LegalEntityName          string
	StoreName                string
	BusinessLicenseNumber    string
	FoodPermitNumber         string
	RegisteredAddress        string
	OperatingAddress         string
	BusinessLicenseIssuedAt  time.Time
	BusinessLicenseExpiresAt time.Time
	FoodPermitIssuedAt       time.Time
	FoodPermitExpiresAt      time.Time
	BusinessScope            []string
	Documents                []QualificationDocument
}

type SubmitQualificationInput struct {
	ActorID     string
	EvidenceIDs []string
}

type ReviewQualificationInput struct {
	ReviewerID  string
	Status      QualificationStatus
	Reason      string
	EvidenceIDs []string
}

type SiteInspectionInput struct {
	InspectorID string
	Result      SiteInspectionResult
	Notes       string
	EvidenceIDs []string
	InspectedAt time.Time
}

type CreateBatchInput struct {
	MerchantID       string
	CreatedBy        string
	ProductID        string
	CampaignID       string
	ProductionDate   time.Time
	ShelfLifeMinutes int
	StorageCondition string
	QuantityPlanned  int
	UnitWeightGrams  int
	Specification    ProductSpec
	IngredientLots   []IngredientLot
}

type RecordProductionInput struct {
	ActorID          string
	Quantity         int
	ProducedAt       time.Time
	ExpiresAt        time.Time
	ShelfLifeMinutes int
}

type TransitionInput struct {
	ActorID     string
	Reason      string
	EvidenceIDs []string
}

type AssociateOrdersInput struct {
	ActorID string
	Orders  []OrderAssociationInput
}

type OrderAssociationInput struct {
	OrderID  string
	Quantity int
}

type CreateIncidentInput struct {
	MerchantID       string
	ReportedBy       string
	Category         string
	Severity         IncidentSeverity
	Title            string
	Description      string
	BatchIDs         []string
	OrderIDs         []string
	RegulatoryReport RegulatoryReport
	EvidenceIDs      []string
	ReportedAt       time.Time
}

type IncidentTransitionInput struct {
	TransitionInput
	ContainmentAction    string
	InvestigationSummary string
	ResolutionSummary    string
	RegulatoryReport     *RegulatoryReport
}

type CreateRecallInput struct {
	IncidentID       string
	MerchantID       string
	Scope            RecallScope
	Reason           string
	BatchIDs         []string
	OrderIDs         []string
	AffectedQuantity int
	EvidenceIDs      []string
}

type RecallTransitionInput struct {
	TransitionInput
	RecoveredQuantity int
	DisposedQuantity  int
	NotifiedAt        *time.Time
}

type CreateDispositionInput struct {
	IncidentID  string
	RecallID    string
	BatchID     string
	OrderID     string
	Type        DispositionType
	Quantity    int
	Unit        string
	ActionBy    string
	Notes       string
	EvidenceIDs []string
}

type AddEvidenceInput struct {
	Kind        EvidenceKind
	URI         string
	SHA256      string
	MimeType    string
	CapturedBy  string
	CapturedAt  time.Time
	Description string
	Metadata    map[string]string
}

type Service interface {
	CreateQualification(context.Context, CreateQualificationInput) (MerchantQualification, error)
	GetQualification(context.Context, string) (MerchantQualification, error)
	ListQualifications(context.Context, string, QualificationStatus) ([]MerchantQualification, error)
	SubmitQualification(context.Context, string, SubmitQualificationInput) (MerchantQualification, error)
	ReviewQualification(context.Context, string, ReviewQualificationInput) (MerchantQualification, error)
	RecordSiteInspection(context.Context, string, SiteInspectionInput) (MerchantQualification, error)

	CreateBatch(context.Context, CreateBatchInput) (FoodBatch, error)
	GetBatch(context.Context, string) (FoodBatch, error)
	ListBatches(context.Context, string, FoodBatchStatus) ([]FoodBatch, error)
	RecordBatchProduction(context.Context, string, RecordProductionInput) (FoodBatch, error)
	AssociateOrders(context.Context, string, AssociateOrdersInput) (FoodBatch, error)
	TransitionBatch(context.Context, string, FoodBatchStatus, TransitionInput) (FoodBatch, error)

	CreateIncident(context.Context, CreateIncidentInput) (FoodSafetyIncident, error)
	GetIncident(context.Context, string) (FoodSafetyIncident, error)
	ListIncidents(context.Context, string, IncidentStatus) ([]FoodSafetyIncident, error)
	TransitionIncident(context.Context, string, IncidentStatus, IncidentTransitionInput) (FoodSafetyIncident, error)

	CreateRecall(context.Context, CreateRecallInput) (Recall, error)
	GetRecall(context.Context, string) (Recall, error)
	ListRecalls(context.Context, string, RecallStatus) ([]Recall, error)
	TransitionRecall(context.Context, string, RecallStatus, RecallTransitionInput) (Recall, error)

	CreateDisposition(context.Context, CreateDispositionInput) (Disposition, error)
	GetDisposition(context.Context, string) (Disposition, error)
	TransitionDisposition(context.Context, string, DispositionStatus, TransitionInput) (Disposition, error)

	AddEvidence(context.Context, EvidenceSubject, AddEvidenceInput) (Evidence, error)
	ListEvidence(context.Context, EvidenceSubject) ([]Evidence, error)
	VerifyEvidenceChain(context.Context, EvidenceSubject) error
}
