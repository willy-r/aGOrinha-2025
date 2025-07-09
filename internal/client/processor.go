package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gorinha-2025/internal/core"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type PaymentClient struct {
	defaultURL  string
	fallbackURL string
	http        *http.Client

	healthMu   sync.Mutex
	lastCheck  time.Time
	defaultUp  bool
	retryDelay time.Duration
}

func NewPaymentClient(defaultURL, fallbackURL string) *PaymentClient {
	return &PaymentClient{
		defaultURL:  defaultURL,
		fallbackURL: fallbackURL,
		http: &http.Client{
			Timeout: 5 * time.Second,
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
	resp, err := pc.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("client error: %d", resp.StatusCode)
	}

	io.Copy(io.Discard, resp.Body)
	return nil
}

func (pc *PaymentClient) checkDefaultHealth() bool {
	pc.healthMu.Lock()
	defer pc.healthMu.Unlock()

	now := time.Now()
	if now.Before(pc.lastCheck.Add(pc.retryDelay)) {
		return pc.defaultUp
	}

	resp, err := pc.http.Get(pc.defaultURL + "/payments/service-health")
	pc.lastCheck = now

	if err != nil {
		pc.defaultUp = false
		pc.retryDelay = 5 * time.Second // Default fallback
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		pc.defaultUp = false
		retryAfter := resp.Header.Get("Retry-After")
		if sec, err := strconv.Atoi(retryAfter); err == nil {
			pc.retryDelay = time.Duration(sec) * time.Second
		} else {
			pc.retryDelay = 5 * time.Second
		}
		return false
	}

	if resp.StatusCode != 200 {
		pc.defaultUp = false
		pc.retryDelay = 5 * time.Second
		return false
	}

	var res struct {
		Failing bool `json:"failing"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)

	pc.defaultUp = !res.Failing
	pc.retryDelay = 5 * time.Second
	return pc.defaultUp
}
