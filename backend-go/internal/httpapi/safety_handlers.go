package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/domain"
	"github.com/X-Colder/chihuo/backend-go/internal/safety"
)

type createSafetyIncidentRequest struct {
	MerchantID       string                  `json:"merchant_id,omitempty"`
	Category         string                  `json:"category"`
	Severity         safety.IncidentSeverity `json:"severity"`
	Title            string                  `json:"title"`
	Description      string                  `json:"description"`
	BatchIDs         []string                `json:"batch_ids"`
	OrderIDs         []string                `json:"order_ids"`
	RegulatoryReport safety.RegulatoryReport `json:"regulatory_report"`
	EvidenceIDs      []string                `json:"evidence_ids"`
}

type transitionSafetyIncidentRequest struct {
	Status               safety.IncidentStatus    `json:"status"`
	Reason               string                   `json:"reason"`
	ContainmentAction    string                   `json:"containment_action"`
	InvestigationSummary string                   `json:"investigation_summary"`
	ResolutionSummary    string                   `json:"resolution_summary"`
	RegulatoryReport     *safety.RegulatoryReport `json:"regulatory_report,omitempty"`
	EvidenceIDs          []string                 `json:"evidence_ids"`
}

func (s *Server) handleCreateSafetyIncident(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	s.executeMutation(w, r, current.ID, body, func() (int, any, error) {
		var input createSafetyIncidentRequest
		if err := decodeJSON(body, &input); err != nil {
			return 0, nil, err
		}
		merchantID := strings.TrimSpace(input.MerchantID)
		if current.Role == domain.RoleMerchant {
			merchantID = current.MerchantID
		}
		if merchantID == "" {
			return 0, nil, newRequestError(http.StatusBadRequest, "MERCHANT_REQUIRED", "merchant_id is required", nil)
		}
		if !input.Severity.Valid() {
			return 0, nil, newRequestError(http.StatusBadRequest, "INVALID_SEVERITY", "severity is invalid", nil)
		}
		if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Description) == "" {
			return 0, nil, newRequestError(http.StatusBadRequest, "INVALID_INCIDENT", "title and description are required", nil)
		}
		incident, err := s.safetyService.CreateIncident(r.Context(), safety.CreateIncidentInput{
			MerchantID:       merchantID,
			ReportedBy:       current.ID,
			Category:         strings.TrimSpace(input.Category),
			Severity:         input.Severity,
			Title:            strings.TrimSpace(input.Title),
			Description:      strings.TrimSpace(input.Description),
			BatchIDs:         input.BatchIDs,
			OrderIDs:         input.OrderIDs,
			RegulatoryReport: input.RegulatoryReport,
			EvidenceIDs:      input.EvidenceIDs,
			ReportedAt:       time.Now().UTC(),
		})
		if err != nil {
			return 0, nil, err
		}
		return http.StatusCreated, incident, nil
	})
}

func (s *Server) handleListSafetyIncidents(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	merchantID := current.MerchantID
	if current.Role == domain.RoleAdmin {
		merchantID = strings.TrimSpace(r.URL.Query().Get("merchant_id"))
	}
	status := safety.IncidentStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	incidents, err := s.safetyService.ListIncidents(r.Context(), merchantID, status)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, incidents)
}

func (s *Server) handleTransitionSafetyIncident(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	var input transitionSafetyIncidentRequest
	if err := decodeJSON(body, &input); err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	if !input.Status.Valid() {
		s.writeOperationError(w, r, newRequestError(http.StatusBadRequest, "INVALID_STATUS", "status is invalid", nil))
		return
	}
	incident, err := s.safetyService.GetIncident(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	if current.Role == domain.RoleMerchant && incident.MerchantID != current.MerchantID {
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "incident does not belong to the current merchant", nil)
		return
	}
	updated, err := s.safetyService.TransitionIncident(r.Context(), incident.ID, input.Status, safety.IncidentTransitionInput{
		TransitionInput: safety.TransitionInput{
			ActorID:     current.ID,
			Reason:      input.Reason,
			EvidenceIDs: input.EvidenceIDs,
		},
		ContainmentAction:    input.ContainmentAction,
		InvestigationSummary: input.InvestigationSummary,
		ResolutionSummary:    input.ResolutionSummary,
		RegulatoryReport:     input.RegulatoryReport,
	})
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, updated)
}
