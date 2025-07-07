package core

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type PaymentRequest struct {
	CorrelationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
	RequestedAt   string  `json:"requestedAt"` // ISO string
}

var (
	ErrInvalidUUID   = errors.New("invalid UUID")
	ErrInvalidAmount = errors.New("amount must be greater than 0")
	ErrInvalidBody   = errors.New("invalid JSON body")
)

func ParseAndValidatePayment(body []byte) (*PaymentRequest, error) {
	var req PaymentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, ErrInvalidBody
	}

	if _, err := uuid.Parse(req.CorrelationID); err != nil {
		return nil, ErrInvalidUUID
	}

	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	req.RequestedAt = time.Now().UTC().Format(time.RFC3339Nano)

	return &req, nil
}
