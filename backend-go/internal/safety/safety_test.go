package safety

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMemoryServiceSafetyLifecycle(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	service := NewMemoryServiceWithClock(func() time.Time { return now })

	qualification, err := service.CreateQualification(ctx, CreateQualificationInput{
		MerchantID:               "merchant-1",
		CreatedBy:                "merchant-user",
		LegalEntityName:          "持火餐饮有限公司",
		StoreName:                "园区厨房",
		BusinessLicenseNumber:    "LIC-001",
		FoodPermitNumber:         "FOOD-001",
		RegisteredAddress:        "上海市浦东新区登记地址",
		OperatingAddress:         "上海市浦东新区经营地址",
		BusinessLicenseIssuedAt:  now.Add(-365 * 24 * time.Hour),
		BusinessLicenseExpiresAt: now.Add(365 * 24 * time.Hour),
		FoodPermitIssuedAt:       now.Add(-180 * 24 * time.Hour),
		FoodPermitExpiresAt:      now.Add(180 * 24 * time.Hour),
		BusinessScope:            []string{"餐饮服务"},
		Documents: []QualificationDocument{
			{
				Kind:      DocumentBusinessLicense,
				URI:       "https://storage.example/license.pdf",
				SHA256:    repeatedHex("a"),
				IssuedAt:  now.Add(-365 * 24 * time.Hour),
				ExpiresAt: timePtr(now.Add(365 * 24 * time.Hour)),
			},
			{
				Kind:      DocumentFoodPermit,
				URI:       "https://storage.example/food-permit.pdf",
				SHA256:    repeatedHex("b"),
				IssuedAt:  now.Add(-180 * 24 * time.Hour),
				ExpiresAt: timePtr(now.Add(180 * 24 * time.Hour)),
			},
		},
	})
	if err != nil {
		t.Fatalf("create qualification: %v", err)
	}

	qualificationEvidence, err := service.AddEvidence(ctx, EvidenceSubject{
		Type: EvidenceQualification,
		ID:   qualification.ID,
	}, AddEvidenceInput{
		Kind:        EvidenceDocument,
		URI:         "https://storage.example/site-review.pdf",
		SHA256:      repeatedHex("c"),
		CapturedBy:  "reviewer-1",
		CapturedAt:  now,
		Description: "资质初审材料",
		Metadata:    map[string]string{"source": "site-review", "version": "1"},
	})
	if err != nil {
		t.Fatalf("add qualification evidence: %v", err)
	}
	if err := service.VerifyEvidenceChain(ctx, EvidenceSubject{Type: EvidenceQualification, ID: qualification.ID}); err != nil {
		t.Fatalf("verify qualification evidence chain: %v", err)
	}

	qualification, err = service.RecordSiteInspection(ctx, qualification.ID, SiteInspectionInput{
		InspectorID: "inspector-1",
		Result:      SiteInspectionPassed,
		Notes:       "加工区、冷藏区和包装区符合检查要求",
		EvidenceIDs: []string{qualificationEvidence.ID},
	})
	if err != nil {
		t.Fatalf("record site inspection: %v", err)
	}
	qualification, err = service.SubmitQualification(ctx, qualification.ID, SubmitQualificationInput{
		ActorID:     "merchant-user",
		EvidenceIDs: []string{qualificationEvidence.ID},
	})
	if err != nil {
		t.Fatalf("submit qualification: %v", err)
	}
	if qualification.Status != QualificationPendingReview || len(qualification.StatusHistory) != 1 {
		t.Fatalf("unexpected submitted qualification: %+v", qualification)
	}
	qualification, err = service.ReviewQualification(ctx, qualification.ID, ReviewQualificationInput{
		ReviewerID:  "admin-1",
		Status:      QualificationApproved,
		Reason:      "资质和现场检查通过",
		EvidenceIDs: []string{qualificationEvidence.ID},
	})
	if err != nil {
		t.Fatalf("approve qualification: %v", err)
	}
	if qualification.Status != QualificationApproved || len(qualification.ReviewHistory) != 1 {
		t.Fatalf("unexpected approved qualification: %+v", qualification)
	}

	batch, err := service.CreateBatch(ctx, CreateBatchInput{
		MerchantID:       "merchant-1",
		CreatedBy:        "food-safety-manager",
		ProductID:        "product-low-oil-chicken-rice",
		CampaignID:       "campaign-1",
		ProductionDate:   now,
		ShelfLifeMinutes: 180,
		StorageCondition: "常温不超过两小时，超过后冷藏",
		QuantityPlanned:  10,
		UnitWeightGrams:  350,
		Specification: ProductSpec{
			WeightGrams:         350,
			Ingredients:         []string{"鸡肉", "米饭", "西兰花"},
			Allergens:           []string{"不含花生"},
			OilLevel:            "LOW",
			SaltLevel:           "LOW",
			StorageInstructions: "常温不超过两小时",
		},
		IngredientLots: []IngredientLot{
			{
				Ingredient:    "鸡肉",
				Supplier:      "供应商A",
				LotNumber:     "CHICKEN-20260902",
				ReceivedAt:    now.Add(-2 * time.Hour),
				ExpiresAt:     now.Add(48 * time.Hour),
				QuantityGrams: 5000,
			},
		},
	})
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	batch, err = service.TransitionBatch(ctx, batch.ID, BatchScheduled, TransitionInput{ActorID: "food-safety-manager"})
	if err != nil {
		t.Fatalf("schedule batch: %v", err)
	}
	batch, err = service.TransitionBatch(ctx, batch.ID, BatchProducing, TransitionInput{ActorID: "food-safety-manager"})
	if err != nil {
		t.Fatalf("start batch production: %v", err)
	}
	batch, err = service.RecordBatchProduction(ctx, batch.ID, RecordProductionInput{
		ActorID:    "food-safety-manager",
		Quantity:   2,
		ProducedAt: now.Add(30 * time.Minute),
		ExpiresAt:  now.Add(210 * time.Minute),
	})
	if err != nil {
		t.Fatalf("record batch production: %v", err)
	}
	if batch.QuantityProduced != 2 || batch.QuantityRemaining != 2 {
		t.Fatalf("unexpected production quantity: %+v", batch)
	}
	batch, err = service.AssociateOrders(ctx, batch.ID, AssociateOrdersInput{
		ActorID: "food-safety-manager",
		Orders: []OrderAssociationInput{
			{OrderID: "order-1", Quantity: 1},
			{OrderID: "order-2", Quantity: 1},
		},
	})
	if err != nil {
		t.Fatalf("associate orders: %v", err)
	}
	batch, err = service.TransitionBatch(ctx, batch.ID, BatchPacked, TransitionInput{ActorID: "food-safety-manager"})
	if err != nil {
		t.Fatalf("pack batch: %v", err)
	}
	batch, err = service.TransitionBatch(ctx, batch.ID, BatchReadyForHandoff, TransitionInput{ActorID: "food-safety-manager"})
	if err != nil {
		t.Fatalf("ready batch: %v", err)
	}
	batch, err = service.TransitionBatch(ctx, batch.ID, BatchInDelivery, TransitionInput{ActorID: "rider-1"})
	if err != nil {
		t.Fatalf("deliver batch: %v", err)
	}
	batch, err = service.TransitionBatch(ctx, batch.ID, BatchCompleted, TransitionInput{ActorID: "rider-1"})
	if err != nil {
		t.Fatalf("complete batch: %v", err)
	}
	batch.Orders[0].OrderID = "mutated-outside-service"
	storedBatch, err := service.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("get batch: %v", err)
	}
	if storedBatch.Orders[0].OrderID != "order-1" {
		t.Fatal("memory service returned mutable internal order associations")
	}

	incident, err := service.CreateIncident(ctx, CreateIncidentInput{
		MerchantID:  "merchant-1",
		ReportedBy:  "consumer-support",
		Category:    "疑似食品安全",
		Severity:    SeverityHigh,
		Title:       "消费者反馈疑似异物",
		Description: "两笔订单反馈同一生产批次存在疑似异物，需要立即隔离调查。",
		BatchIDs:    []string{batch.ID},
		OrderIDs:    []string{"order-1", "order-2"},
	})
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	incident, err = service.TransitionIncident(ctx, incident.ID, IncidentContained, IncidentTransitionInput{
		TransitionInput:   TransitionInput{ActorID: "admin-1", Reason: "先行隔离涉事批次"},
		ContainmentAction: "暂停销售并隔离涉事生产批次",
	})
	if err != nil {
		t.Fatalf("contain incident: %v", err)
	}
	quarantined, err := service.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("get quarantined batch: %v", err)
	}
	if quarantined.Status != BatchQuarantined {
		t.Fatalf("batch status after containment = %s", quarantined.Status)
	}
	incident, err = service.TransitionIncident(ctx, incident.ID, IncidentInvestigating, IncidentTransitionInput{
		TransitionInput:      TransitionInput{ActorID: "investigator-1"},
		InvestigationSummary: "核对原料批次、生产记录和配送记录",
	})
	if err != nil {
		t.Fatalf("investigate incident: %v", err)
	}
	incident, err = service.TransitionIncident(ctx, incident.ID, IncidentConfirmed, IncidentTransitionInput{
		TransitionInput: TransitionInput{ActorID: "investigator-1", Reason: "调查确认涉事批次需要召回"},
	})
	if err != nil {
		t.Fatalf("confirm incident: %v", err)
	}

	recall, err := service.CreateRecall(ctx, CreateRecallInput{
		IncidentID:       incident.ID,
		Scope:            RecallBatch,
		Reason:           "涉事批次存在食品安全风险",
		BatchIDs:         []string{batch.ID},
		AffectedQuantity: 2,
	})
	if err != nil {
		t.Fatalf("create recall: %v", err)
	}
	recallEvidence, err := service.AddEvidence(ctx, EvidenceSubject{Type: EvidenceRecall, ID: recall.ID}, AddEvidenceInput{
		Kind:       EvidenceLabReport,
		URI:        "https://storage.example/lab-report.pdf",
		SHA256:     repeatedHex("d"),
		CapturedBy: "investigator-1",
		CapturedAt: now,
	})
	if err != nil {
		t.Fatalf("add recall evidence: %v", err)
	}
	recall, err = service.TransitionRecall(ctx, recall.ID, RecallInitiated, RecallTransitionInput{
		TransitionInput: TransitionInput{
			ActorID:     "admin-1",
			Reason:      "启动批次召回",
			EvidenceIDs: []string{recallEvidence.ID},
		},
	})
	if err != nil {
		t.Fatalf("initiate recall: %v", err)
	}
	if recall.Status != RecallInitiated {
		t.Fatalf("recall status = %s", recall.Status)
	}
	recalledBatch, err := service.GetBatch(ctx, batch.ID)
	if err != nil {
		t.Fatalf("get recalled batch: %v", err)
	}
	if recalledBatch.Status != BatchRecalled || !containsString(recalledBatch.RecallIDs, recall.ID) {
		t.Fatalf("batch was not marked recalled: %+v", recalledBatch)
	}

	recall, err = service.TransitionRecall(ctx, recall.ID, RecallInProgress, RecallTransitionInput{
		TransitionInput: TransitionInput{ActorID: "consumer-support"},
	})
	if err != nil {
		t.Fatalf("start recall execution: %v", err)
	}
	disposition, err := service.CreateDisposition(ctx, CreateDispositionInput{
		IncidentID: incident.ID,
		RecallID:   recall.ID,
		BatchID:    batch.ID,
		Type:       DispositionDestroy,
		Quantity:   1,
		Unit:       "份",
		ActionBy:   "merchant-operator",
		Notes:      "销毁疑似受污染餐品",
	})
	if err != nil {
		t.Fatalf("create disposition: %v", err)
	}
	dispositionEvidence, err := service.AddEvidence(ctx, EvidenceSubject{Type: EvidenceDisposition, ID: disposition.ID}, AddEvidenceInput{
		Kind:       EvidencePhoto,
		URI:        "https://storage.example/destruction.jpg",
		SHA256:     repeatedHex("e"),
		CapturedBy: "merchant-operator",
		CapturedAt: now,
	})
	if err != nil {
		t.Fatalf("add disposition evidence: %v", err)
	}
	_, err = service.TransitionDisposition(ctx, disposition.ID, DispositionCompleted, TransitionInput{
		ActorID:     "merchant-operator",
		EvidenceIDs: []string{dispositionEvidence.ID},
	})
	if err != nil {
		t.Fatalf("complete disposition: %v", err)
	}
	recall, err = service.TransitionRecall(ctx, recall.ID, RecallCompleted, RecallTransitionInput{
		TransitionInput:  TransitionInput{ActorID: "admin-1"},
		DisposedQuantity: 1,
	})
	if err != nil {
		t.Fatalf("complete recall: %v", err)
	}
	if recall.DisposedQuantity != 1 {
		t.Fatalf("disposed quantity = %d", recall.DisposedQuantity)
	}
	incident, err = service.TransitionIncident(ctx, incident.ID, IncidentCompensating, IncidentTransitionInput{
		TransitionInput: TransitionInput{ActorID: "consumer-support"},
	})
	if err != nil {
		t.Fatalf("start compensation: %v", err)
	}
	incident, err = service.TransitionIncident(ctx, incident.ID, IncidentResolved, IncidentTransitionInput{
		TransitionInput:   TransitionInput{ActorID: "admin-1"},
		ResolutionSummary: "已完成批次召回、销毁和消费者处置",
	})
	if err != nil {
		t.Fatalf("resolve incident: %v", err)
	}
	incident, err = service.TransitionIncident(ctx, incident.ID, IncidentClosed, IncidentTransitionInput{
		TransitionInput: TransitionInput{ActorID: "admin-1", Reason: "证据、召回和处置记录已归档"},
	})
	if err != nil {
		t.Fatalf("close incident: %v", err)
	}
	if incident.Status != IncidentClosed || len(incident.StatusHistory) != 7 {
		t.Fatalf("unexpected closed incident: %+v", incident)
	}
	if err := service.VerifyEvidenceChain(ctx, EvidenceSubject{Type: EvidenceDisposition, ID: disposition.ID}); err != nil {
		t.Fatalf("verify disposition evidence chain: %v", err)
	}
}

func TestMemoryServiceRejectsUnsafeTransitionsAndConflicts(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	service := NewMemoryServiceWithClock(func() time.Time { return now })

	_, err := service.CreateBatch(ctx, CreateBatchInput{
		MerchantID:       "merchant-without-approval",
		CreatedBy:        "operator",
		ProductID:        "product-1",
		ProductionDate:   now,
		ShelfLifeMinutes: 60,
		StorageCondition: "常温",
		QuantityPlanned:  1,
		UnitWeightGrams:  300,
		Specification: ProductSpec{
			WeightGrams: 300,
			Ingredients: []string{"食材A"},
		},
	})
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("create batch without approval error = %v", err)
	}

	if CanTransitionIncident(IncidentClosed, IncidentReported) {
		t.Fatal("closed incident must be terminal")
	}
	if CanTransitionRecall(RecallCompleted, RecallInProgress) {
		t.Fatal("completed recall must be terminal")
	}

	qualification := mustCreateApprovedQualification(t, service, now, "merchant-2")
	batch, err := service.CreateBatch(ctx, CreateBatchInput{
		MerchantID:       qualification.MerchantID,
		CreatedBy:        "operator",
		ProductID:        "product-2",
		ProductionDate:   now,
		ShelfLifeMinutes: 60,
		StorageCondition: "冷藏",
		QuantityPlanned:  1,
		UnitWeightGrams:  300,
		Specification: ProductSpec{
			WeightGrams: 300,
			Ingredients: []string{"食材A"},
		},
	})
	if err != nil {
		t.Fatalf("create approved batch: %v", err)
	}
	_, err = service.TransitionBatch(ctx, batch.ID, BatchCompleted, TransitionInput{ActorID: "operator"})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("invalid batch transition error = %v", err)
	}
	_, err = service.AssociateOrders(ctx, batch.ID, AssociateOrdersInput{
		ActorID: "operator",
		Orders:  []OrderAssociationInput{{OrderID: "order-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("associate first order: %v", err)
	}
	_, err = service.AssociateOrders(ctx, batch.ID, AssociateOrdersInput{
		ActorID: "operator",
		Orders:  []OrderAssociationInput{{OrderID: "order-1", Quantity: 1}},
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate order association error = %v", err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = service.GetBatch(cancelled, batch.ID)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context error = %v", err)
	}
}

func TestMemoryServiceConcurrentEvidenceAppend(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	service := NewMemoryServiceWithClock(func() time.Time { return now })
	qualification := mustCreateApprovedQualification(t, service, now, "merchant-3")

	const evidenceCount = 16
	var wait sync.WaitGroup
	errs := make(chan error, evidenceCount)
	for i := 0; i < evidenceCount; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			_, err := service.AddEvidence(ctx, EvidenceSubject{
				Type: EvidenceQualification,
				ID:   qualification.ID,
			}, AddEvidenceInput{
				Kind:       EvidenceRecord,
				URI:        "https://storage.example/record-" + string(rune('a'+index)) + ".json",
				SHA256:     repeatedHex("f"),
				CapturedBy: "system",
				CapturedAt: now,
			})
			errs <- err
		}(i)
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent evidence append: %v", err)
		}
	}
	evidence, err := service.ListEvidence(ctx, EvidenceSubject{Type: EvidenceQualification, ID: qualification.ID})
	if err != nil {
		t.Fatalf("list evidence: %v", err)
	}
	if len(evidence) != evidenceCount {
		t.Fatalf("evidence count = %d, want %d", len(evidence), evidenceCount)
	}
	if err := service.VerifyEvidenceChain(ctx, EvidenceSubject{Type: EvidenceQualification, ID: qualification.ID}); err != nil {
		t.Fatalf("verify concurrent evidence chain: %v", err)
	}
}

func mustCreateApprovedQualification(t *testing.T, service *MemoryService, now time.Time, merchantID string) MerchantQualification {
	t.Helper()
	qualification, err := service.CreateQualification(context.Background(), CreateQualificationInput{
		MerchantID:               merchantID,
		CreatedBy:                "merchant-user",
		LegalEntityName:          "餐饮有限公司",
		StoreName:                "测试厨房",
		BusinessLicenseNumber:    "LIC-" + merchantID,
		FoodPermitNumber:         "FOOD-" + merchantID,
		RegisteredAddress:        "登记地址",
		OperatingAddress:         "经营地址",
		BusinessLicenseIssuedAt:  now.Add(-24 * time.Hour),
		BusinessLicenseExpiresAt: now.Add(365 * 24 * time.Hour),
		FoodPermitIssuedAt:       now.Add(-24 * time.Hour),
		FoodPermitExpiresAt:      now.Add(365 * 24 * time.Hour),
		Documents: []QualificationDocument{
			{Kind: DocumentBusinessLicense, URI: "https://storage.example/license", SHA256: repeatedHex("1"), IssuedAt: now},
			{Kind: DocumentFoodPermit, URI: "https://storage.example/permit", SHA256: repeatedHex("2"), IssuedAt: now},
		},
	})
	if err != nil {
		t.Fatalf("create fixture qualification: %v", err)
	}
	qualification, err = service.RecordSiteInspection(context.Background(), qualification.ID, SiteInspectionInput{
		InspectorID: "inspector",
		Result:      SiteInspectionPassed,
	})
	if err != nil {
		t.Fatalf("inspect fixture qualification: %v", err)
	}
	qualification, err = service.SubmitQualification(context.Background(), qualification.ID, SubmitQualificationInput{
		ActorID: "merchant-user",
	})
	if err != nil {
		t.Fatalf("submit fixture qualification: %v", err)
	}
	qualification, err = service.ReviewQualification(context.Background(), qualification.ID, ReviewQualificationInput{
		ReviewerID: "admin",
		Status:     QualificationApproved,
	})
	if err != nil {
		t.Fatalf("approve fixture qualification: %v", err)
	}
	return qualification
}

func repeatedHex(value string) string {
	result := make([]byte, 64)
	for i := range result {
		result[i] = value[0]
	}
	return string(result)
}
