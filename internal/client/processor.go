package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"gorinha-2025/internal/core"
	"io"
	"net/http"
	"sync"
	"time"
)

type PaymentClient struct {
	defaultURL  string
	fallbackURL string
	http        *http.Client

	healthMu  sync.Mutex
	lastCheck time.Time
	defaultUp bool
}

func NewPaymentClient(defaultURL, fallbackURL string) *PaymentClient {
	return &PaymentClient{
		defaultURL:  defaultURL,
		fallbackURL: fallbackURL,
		http: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (pc *PaymentClient) SendToDefault(p *core.PaymentRequest) error {
	if !pc.checkDefaultHealth() {
		return errors.New("default processor is down (cached)")
	}
	return pc.send(pc.defaultURL+"/payments", p)
}

func (pc *PaymentClient) SendToFallback(p *core.PaymentRequest) error {
	return pc.send(pc.fallbackURL+"/payments", p)
}

func (pc *PaymentClient) send(url string, p *core.PaymentRequest) error {
	body, _ := json.Marshal(p)
	response, err := pc.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 500 {
		return errors.New("processor failed")
	}

	io.Copy(io.Discard, response.Body)
	return nil
}

func (pc *PaymentClient) checkDefaultHealth() bool {
	pc.healthMu.Lock()
	defer pc.healthMu.Unlock()

	now := time.Now()
	if now.Sub(pc.lastCheck) < 5*time.Second {
		return pc.defaultUp
	}

	response, err := pc.http.Get(pc.defaultURL + "payments/service-health")
	if err != nil || response.StatusCode != 200 {
		pc.defaultUp = false
		pc.lastCheck = now
		return false
	}
	defer response.Body.Close()

	var res struct {
		Failing bool `json:"failing"`
	}
	_ = json.NewDecoder(response.Body).Decode(&res)

	pc.defaultUp = !res.Failing
	pc.lastCheck = now
	return pc.defaultUp
}
