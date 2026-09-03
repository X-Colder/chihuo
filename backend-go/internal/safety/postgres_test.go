package safety

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/store"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestNewSafetyUUID(t *testing.T) {
	first := newSafetyUUID()
	second := newSafetyUUID()
	if !isSafetyUUID(first) || !isSafetyUUID(second) {
		t.Fatalf("generated invalid UUIDs: %q %q", first, second)
	}
	if first == second {
		t.Fatal("generated duplicate UUIDs")
	}
}

func TestPersistedEvidenceMetadataRoundTrip(t *testing.T) {
	now := time.Date(2026, 9, 2, 10, 0, 0, 123000000, time.UTC)
	evidence := Evidence{
		ID:          newSafetyUUID(),
		Subject:     EvidenceSubject{Type: EvidenceIncident, ID: newSafetyUUID()},
		Sequence:    1,
		Kind:        EvidenceRecord,
		URI:         "https://storage.example/record.json",
		SHA256:      repeatedHex("a"),
		MimeType:    "application/json",
		CapturedBy:  newSafetyUUID(),
		CapturedAt:  now,
		Description: "incident record",
		Metadata:    map[string]string{"source": "test", "version": "1"},
		CreatedAt:   now,
	}
	evidence.Hash = evidenceHash(evidence)
	raw, err := marshalPersistedEvidenceMetadata(evidence)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	var decoded Evidence
	var internal persistedEvidenceMetadata
	if err := unmarshalPersistedEvidenceMetadata(raw, &decoded, &internal); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if decoded.MimeType != evidence.MimeType ||
		decoded.Description != evidence.Description ||
		decoded.Metadata["source"] != "test" ||
		!internal.CreatedAt.Equal(now) {
		t.Fatalf("metadata round trip mismatch: decoded=%+v internal=%+v", decoded, internal)
	}
}

func TestPostgresServiceCoreLifecycle(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set POSTGRES_TEST_DATABASE_URL to run PostgreSQL safety integration tests")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if err := store.NewPostgresStore(db).Migrate(ctx); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}

	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	service := NewPostgresServiceWithClock(db, func() time.Time { return now })
	ownerID := newSafetyUUID()
	adminID := newSafetyUUID()
	merchantID := newSafetyUUID()
	createdAt := now.Add(-time.Hour)
	for _, user := range []struct {
		id   string
		name string
		role string
	}{
		{ownerID, "safety owner", "MERCHANT"},
		{adminID, "safety admin", "ADMIN"},
	} {
		_, err := db.ExecContext(ctx, `
			INSERT INTO users (id, name, role, external_key, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, user.id, user.name, user.role, "safety-test-"+user.id, createdAt)
		if err != nil {
			t.Fatalf("insert user %s: %v", user.id, err)
		}
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO merchants (id, owner_user_id, name, status, license, created_at)
		VALUES ($1, $2, $3, 'APPROVED', '{}'::jsonb, $4)
	`, merchantID, ownerID, "safety test merchant", createdAt); err != nil {
		t.Fatalf("insert merchant: %v", err)
	}

	qualification, err := service.CreateQualification(ctx, CreateQualificationInput{
		MerchantID:               merchantID,
		CreatedBy:                ownerID,
		LegalEntityName:          "持火餐饮有限公司",
		StoreName:                "测试厨房",
		BusinessLicenseNumber:    "LIC-" + merchantID,
		FoodPermitNumber:         "FOOD-" + merchantID,
		RegisteredAddress:        "登记地址",
		OperatingAddress:         "经营地址",
		BusinessLicenseIssuedAt:  now.Add(-24 * time.Hour),
		BusinessLicenseExpiresAt: now.Add(365 * 24 * time.Hour),
		FoodPermitIssuedAt:       now.Add(-24 * time.Hour),
		FoodPermitExpiresAt:      now.Add(365 * 24 * time.Hour),
		BusinessScope:            []string{"餐饮服务"},
		Documents: []QualificationDocument{
			{Kind: DocumentBusinessLicense, URI: "https://storage.example/license", SHA256: repeatedHex("1")},
			{Kind: DocumentFoodPermit, URI: "https://storage.example/permit", SHA256: repeatedHex("2")},
		},
	})
	if err != nil {
		t.Fatalf("create qualification: %v", err)
	}
	qualificationEvidence, err := service.AddEvidence(ctx,
		EvidenceSubject{Type: EvidenceQualification, ID: qualification.ID},
		AddEvidenceInput{
			Kind:       EvidenceDocument,
			URI:        "https://storage.example/site-review.pdf",
			SHA256:     repeatedHex("3"),
			CapturedBy: adminID,
			CapturedAt: now,
		})
	if err != nil {
		t.Fatalf("add qualification evidence: %v", err)
	}
	if err := service.VerifyEvidenceChain(ctx, EvidenceSubject{Type: EvidenceQualification, ID: qualification.ID}); err != nil {
		t.Fatalf("verify qualification evidence: %v", err)
	}
	if _, err := service.RecordSiteInspection(ctx, qualification.ID, SiteInspectionInput{
		InspectorID: adminID,
		Result:      SiteInspectionPassed,
		EvidenceIDs: []string{qualificationEvidence.ID},
	}); err != nil {
		t.Fatalf("record site inspection: %v", err)
	}
	if _, err := service.SubmitQualification(ctx, qualification.ID, SubmitQualificationInput{
		ActorID: ownerID,
	}); err != nil {
		t.Fatalf("submit qualification: %v", err)
	}
	qualification, err = service.ReviewQualification(ctx, qualification.ID, ReviewQualificationInput{
		ReviewerID: adminID,
		Status:     QualificationApproved,
		Reason:     "passed",
	})
	if err != nil {
		t.Fatalf("approve qualification: %v", err)
	}
	if qualification.Status != QualificationApproved {
		t.Fatalf("qualification status = %s", qualification.Status)
	}

	productID := newSafetyUUID()
	batch, err := service.CreateBatch(ctx, CreateBatchInput{
		MerchantID:       merchantID,
		CreatedBy:        ownerID,
		ProductID:        productID,
		ProductionDate:   now,
		ShelfLifeMinutes: 180,
		StorageCondition: "冷藏",
		QuantityPlanned:  10,
		UnitWeightGrams:  350,
		Specification: ProductSpec{
			WeightGrams: 350,
			Ingredients: []string{"鸡肉", "米饭"},
		},
		IngredientLots: []IngredientLot{{
			Ingredient:    "鸡肉",
			Supplier:      "供应商A",
			LotNumber:     "LOT-" + merchantID,
			ReceivedAt:    now.Add(-time.Hour),
			ExpiresAt:     now.Add(48 * time.Hour),
			QuantityGrams: 5000,
		}},
	})
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	if _, err := service.TransitionBatch(ctx, batch.ID, BatchScheduled, TransitionInput{ActorID: ownerID}); err != nil {
		t.Fatalf("schedule batch: %v", err)
	}
	if _, err := service.TransitionBatch(ctx, batch.ID, BatchProducing, TransitionInput{ActorID: ownerID}); err != nil {
		t.Fatalf("start batch: %v", err)
	}
	if _, err := service.RecordBatchProduction(ctx, batch.ID, RecordProductionInput{
		ActorID:    ownerID,
		Quantity:   2,
		ProducedAt: now.Add(30 * time.Minute),
		ExpiresAt:  now.Add(210 * time.Minute),
	}); err != nil {
		t.Fatalf("record production: %v", err)
	}
	batchEvidence, err := service.AddEvidence(ctx,
		EvidenceSubject{Type: EvidenceBatch, ID: batch.ID},
		AddEvidenceInput{
			Kind:       EvidenceRecord,
			URI:        "https://storage.example/batch-record.json",
			SHA256:     repeatedHex("4"),
			CapturedBy: ownerID,
			CapturedAt: now,
		})
	if err != nil {
		t.Fatalf("add batch evidence: %v", err)
	}
	if err := service.VerifyEvidenceChain(ctx, EvidenceSubject{Type: EvidenceBatch, ID: batch.ID}); err != nil {
		t.Fatalf("verify batch evidence: %v", err)
	}
	if _, err := service.TransitionBatch(ctx, batch.ID, BatchPacked, TransitionInput{
		ActorID:     ownerID,
		EvidenceIDs: []string{batchEvidence.ID},
	}); err != nil {
		t.Fatalf("pack batch: %v", err)
	}

	incident, err := service.CreateIncident(ctx, CreateIncidentInput{
		MerchantID:  merchantID,
		ReportedBy:  adminID,
		Category:    "疑似食品安全",
		Severity:    SeverityHigh,
		Title:       "测试事件",
		Description: "测试批次需要隔离调查。",
		BatchIDs:    []string{batch.ID},
	})
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	incident, err = service.TransitionIncident(ctx, incident.ID, IncidentContained, IncidentTransitionInput{
		TransitionInput:   TransitionInput{ActorID: adminID, Reason: "隔离"},
		ContainmentAction: "暂停销售并隔离批次",
	})
	if err != nil {
		t.Fatalf("contain incident: %v", err)
	}
	if incident.Status != IncidentContained {
		t.Fatalf("incident status = %s", incident.Status)
	}
	batch, err = service.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("get quarantined batch: %v", err)
	}
	if batch.Status != BatchQuarantined {
		t.Fatalf("batch status after containment = %s", batch.Status)
	}

	const concurrentEvidence = 8
	var wait sync.WaitGroup
	errs := make(chan error, concurrentEvidence)
	for index := 0; index < concurrentEvidence; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			_, err := service.AddEvidence(ctx,
				EvidenceSubject{Type: EvidenceIncident, ID: incident.ID},
				AddEvidenceInput{
					Kind:       EvidenceRecord,
					URI:        fmt.Sprintf("https://storage.example/incident-%d.json", index),
					SHA256:     repeatedHex("5"),
					CapturedBy: adminID,
					CapturedAt: now,
				})
			errs <- err
		}(index)
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent incident evidence: %v", err)
		}
	}
	items, err := service.ListEvidence(ctx, EvidenceSubject{Type: EvidenceIncident, ID: incident.ID})
	if err != nil {
		t.Fatalf("list incident evidence: %v", err)
	}
	if len(items) != concurrentEvidence {
		t.Fatalf("incident evidence count = %d, want %d", len(items), concurrentEvidence)
	}
	if err := service.VerifyEvidenceChain(ctx, EvidenceSubject{Type: EvidenceIncident, ID: incident.ID}); err != nil {
		t.Fatalf("verify incident evidence chain: %v", err)
	}
}
