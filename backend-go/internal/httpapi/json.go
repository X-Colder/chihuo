package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/X-Colder/chihuo/backend-go/internal/domain"
	"github.com/X-Colder/chihuo/backend-go/internal/safety"
	"github.com/X-Colder/chihuo/backend-go/internal/store"
)

const maxJSONBodyBytes = 1 << 20

type apiEnvelope struct {
	Data  any         `json:"data,omitempty"`
	Error *apiProblem `json:"error,omitempty"`
}

type apiProblem struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id"`
}

type requestError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func (e *requestError) Error() string { return e.Message }

func newRequestError(status int, code, message string, details any) error {
	return &requestError{Status: status, Code: code, Message: message, Details: details}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeRawJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, apiEnvelope{Data: data})
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, details any) {
	writeJSON(w, status, apiEnvelope{
		Error: &apiProblem{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: requestID(r.Context()),
		},
	})
}

func readJSONBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBodyBytes+1))
	if err != nil {
		return nil, newRequestError(http.StatusBadRequest, "INVALID_BODY", "request body could not be read", nil)
	}
	if len(body) == 0 {
		return nil, newRequestError(http.StatusBadRequest, "INVALID_BODY", "request body is required", nil)
	}
	if len(body) > maxJSONBodyBytes {
		return nil, newRequestError(http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE", "request body is too large", nil)
	}
	return body, nil
}

func decodeJSON(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return newRequestError(http.StatusBadRequest, "INVALID_JSON", "request body is invalid", err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return newRequestError(http.StatusBadRequest, "INVALID_JSON", "request body must contain one JSON object", nil)
	}
	return nil
}

func (s *Server) executeMutation(w http.ResponseWriter, r *http.Request, actorID string, body []byte, operation func() (int, any, error)) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		status, data, err := operation()
		if err != nil {
			s.writeOperationError(w, r, err)
			return
		}
		writeData(w, status, data)
		return
	}
	if len(key) > 128 {
		writeError(w, r, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key must be at most 128 characters", nil)
		return
	}
	lockKey := actorID + ":" + key
	lock := s.idempotencyLock(lockKey)
	lock.Lock()
	defer lock.Unlock()
	fingerprint := requestFingerprint(r, body)
	record, err := s.store.GetIdempotency(r.Context(), actorID, key)
	switch {
	case err == nil:
		if record.Fingerprint != fingerprint {
			writeError(w, r, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key was already used with a different request", nil)
			return
		}
		writeRawJSON(w, record.Status, record.Response)
		return
	case !errors.Is(err, store.ErrNotFound):
		s.writeOperationError(w, r, err)
		return
	}
	status, data, err := operation()
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	raw, err := json.Marshal(apiEnvelope{Data: data})
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	if err := s.store.PutIdempotency(r.Context(), domainIdempotencyRecord(actorID, key, fingerprint, status, raw)); err != nil {
		if errors.Is(err, store.ErrConflict) {
			writeError(w, r, http.StatusConflict, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key was already used with a different request", nil)
			return
		}
		s.writeOperationError(w, r, err)
		return
	}
	writeRawJSON(w, status, raw)
}

func (s *Server) idempotencyLock(key string) *sync.Mutex {
	value, loaded := s.idempotencyLocks.LoadOrStore(key, &sync.Mutex{})
	if loaded {
		return value.(*sync.Mutex)
	}
	return value.(*sync.Mutex)
}

func domainIdempotencyRecord(actorID, key, fingerprint string, status int, response []byte) domain.IdempotencyRecord {
	return domain.IdempotencyRecord{
		ActorID:     actorID,
		Key:         key,
		Fingerprint: fingerprint,
		Status:      status,
		Response:    response,
		CreatedAt:   time.Now().UTC(),
	}
}

func requestFingerprint(r *http.Request, body []byte) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(r.Method + "\n" + r.URL.Path + "\n"))
	_, _ = hasher.Write(body)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *Server) writeOperationError(w http.ResponseWriter, r *http.Request, err error) {
	var typed *requestError
	if errors.As(err, &typed) {
		writeError(w, r, typed.Status, typed.Code, typed.Message, typed.Details)
		return
	}
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
	case errors.Is(err, store.ErrConflict):
		writeError(w, r, http.StatusConflict, "CONFLICT", "resource conflict", nil)
	case errors.Is(err, store.ErrForbidden):
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "operation is not allowed", nil)
	case errors.Is(err, store.ErrInvalid):
		writeError(w, r, http.StatusConflict, "INVALID_STATE", "resource is not in a valid state for this operation", nil)
	case errors.Is(err, safety.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "SAFETY_NOT_FOUND", "safety resource not found", nil)
	case errors.Is(err, safety.ErrConflict):
		writeError(w, r, http.StatusConflict, "SAFETY_CONFLICT", "safety resource conflict", nil)
	case errors.Is(err, safety.ErrInvalidState), errors.Is(err, safety.ErrInvalidTransition):
		writeError(w, r, http.StatusConflict, "SAFETY_INVALID_STATE", "safety state transition is invalid", nil)
	case errors.Is(err, safety.ErrValidation):
		writeError(w, r, http.StatusBadRequest, "SAFETY_VALIDATION_ERROR", "safety data is invalid", nil)
	default:
		s.logger.Error("operation failed", "request_id", requestID(r.Context()), "error", err)
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func (s *Server) requireStoreReady(w http.ResponseWriter, r *http.Request) bool {
	if err := s.store.Ping(r.Context()); err != nil {
		s.writeOperationError(w, r, err)
		return false
	}
	return true
}
