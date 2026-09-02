package payment

func TransitionPayment(from, to PaymentStatus) error {
	if !from.Valid() || !to.Valid() {
		return &InvalidTransitionError{Aggregate: "payment", From: string(from), To: string(to)}
	}
	if from == to {
		return nil
	}
	allowed := map[PaymentStatus]map[PaymentStatus]struct{}{
		PaymentPending: {
			PaymentPrepayCreated: {},
			PaymentPaying:        {},
			PaymentPaid:          {},
			PaymentClosed:        {},
			PaymentFailed:        {},
		},
		PaymentPrepayCreated: {
			PaymentPaying: {},
			PaymentPaid:   {},
			PaymentClosed: {},
			PaymentFailed: {},
		},
		PaymentPaying: {
			PaymentPaid:   {},
			PaymentClosed: {},
			PaymentFailed: {},
		},
		PaymentPaid: {
			PaymentPartiallyRefunded: {},
			PaymentRefunded:          {},
		},
		PaymentPartiallyRefunded: {
			PaymentPartiallyRefunded: {},
			PaymentRefunded:          {},
		},
	}
	if _, ok := allowed[from][to]; ok {
		return nil
	}
	return &InvalidTransitionError{Aggregate: "payment", From: string(from), To: string(to)}
}

func ApplyPaymentEvent(current PaymentStatus, event PaymentEvent) (PaymentStatus, error) {
	var next PaymentStatus
	switch event {
	case PaymentEventPrepayCreated:
		next = PaymentPrepayCreated
	case PaymentEventPaymentPending:
		next = PaymentPending
	case PaymentEventPaymentPaying:
		next = PaymentPaying
	case PaymentEventPaymentSucceeded:
		next = PaymentPaid
	case PaymentEventPaymentClosed:
		next = PaymentClosed
	case PaymentEventPaymentFailed:
		next = PaymentFailed
	case PaymentEventRefundPartially:
		next = PaymentPartiallyRefunded
	case PaymentEventRefundSucceeded:
		next = PaymentRefunded
	default:
		return current, &InvalidTransitionError{Aggregate: "payment", From: string(current), To: string(event)}
	}
	if err := TransitionPayment(current, next); err != nil {
		return current, err
	}
	return next, nil
}

func TransitionRefund(from, to RefundStatus) error {
	if !from.Valid() || !to.Valid() {
		return &InvalidTransitionError{Aggregate: "refund", From: string(from), To: string(to)}
	}
	if from == to {
		return nil
	}
	allowed := map[RefundStatus]map[RefundStatus]struct{}{
		RefundPending: {
			RefundProcessing: {},
			RefundSucceeded:  {},
			RefundFailed:     {},
			RefundClosed:     {},
		},
		RefundProcessing: {
			RefundSucceeded: {},
			RefundFailed:    {},
			RefundClosed:    {},
		},
		RefundFailed: {
			RefundProcessing: {},
			RefundSucceeded:  {},
		},
	}
	if _, ok := allowed[from][to]; ok {
		return nil
	}
	return &InvalidTransitionError{Aggregate: "refund", From: string(from), To: string(to)}
}

func ApplyRefundEvent(current RefundStatus, event RefundEvent) (RefundStatus, error) {
	var next RefundStatus
	switch event {
	case RefundEventProcessing:
		next = RefundProcessing
	case RefundEventSucceeded:
		next = RefundSucceeded
	case RefundEventFailed:
		next = RefundFailed
	case RefundEventClosed:
		next = RefundClosed
	default:
		return current, &InvalidTransitionError{Aggregate: "refund", From: string(current), To: string(event)}
	}
	if err := TransitionRefund(current, next); err != nil {
		return current, err
	}
	return next, nil
}
