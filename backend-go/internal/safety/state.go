package safety

func CanTransitionQualification(from, to QualificationStatus) bool {
	switch from {
	case QualificationDraft:
		return to == QualificationPendingReview
	case QualificationPendingReview:
		return to == QualificationApproved || to == QualificationRejected
	case QualificationRejected, QualificationSuspended, QualificationExpired:
		return to == QualificationPendingReview
	case QualificationApproved:
		return to == QualificationSuspended || to == QualificationExpired
	default:
		return false
	}
}

func CanTransitionBatch(from, to FoodBatchStatus) bool {
	switch from {
	case BatchDraft:
		return to == BatchScheduled || to == BatchCancelled || to == BatchQuarantined
	case BatchScheduled:
		return to == BatchProducing || to == BatchCancelled || to == BatchQuarantined
	case BatchProducing:
		return to == BatchPacked || to == BatchQuarantined
	case BatchPacked:
		return to == BatchReadyForHandoff || to == BatchQuarantined || to == BatchRecalled
	case BatchReadyForHandoff:
		return to == BatchInDelivery || to == BatchQuarantined || to == BatchRecalled
	case BatchInDelivery:
		return to == BatchCompleted || to == BatchQuarantined || to == BatchRecalled
	case BatchCompleted:
		return to == BatchQuarantined || to == BatchRecalled
	case BatchQuarantined:
		return to == BatchRecalled || to == BatchDisposed
	case BatchRecalled:
		return to == BatchDisposed
	default:
		return false
	}
}

func CanTransitionIncident(from, to IncidentStatus) bool {
	switch from {
	case IncidentReported:
		return to == IncidentContained || to == IncidentInvestigating || to == IncidentDismissed
	case IncidentContained:
		return to == IncidentInvestigating || to == IncidentRecalling || to == IncidentDismissed
	case IncidentInvestigating:
		return to == IncidentContained || to == IncidentConfirmed || to == IncidentDismissed
	case IncidentConfirmed:
		return to == IncidentRecalling || to == IncidentCompensating || to == IncidentResolved
	case IncidentRecalling:
		return to == IncidentCompensating || to == IncidentResolved
	case IncidentCompensating:
		return to == IncidentResolved
	case IncidentResolved:
		return to == IncidentClosed
	case IncidentDismissed:
		return to == IncidentClosed
	default:
		return false
	}
}

func CanTransitionRecall(from, to RecallStatus) bool {
	switch from {
	case RecallDraft:
		return to == RecallInitiated || to == RecallCancelled
	case RecallInitiated:
		return to == RecallInProgress
	case RecallInProgress:
		return to == RecallCompleted
	default:
		return false
	}
}

func CanTransitionDisposition(from, to DispositionStatus) bool {
	switch from {
	case DispositionPlanned:
		return to == DispositionCompleted || to == DispositionFailed || to == DispositionCancelled
	default:
		return false
	}
}
