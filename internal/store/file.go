package store

import (
	"encoding/json"
	"gorinha-2025/internal/config"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

var filePath = config.GetEnv("STORE_JSON_PATH", "/data/store.json")
var lockPath = config.GetEnv("STORE_LOCK_PATH", "/data/store.lock")

type fileRecord struct {
	Processor   string    `json:"processor"`
	Amount      float64   `json:"amount"`
	RequestedAt time.Time `json:"requestedAt"`
}

type Summary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

func AddPaymentToFile(processor string, amount float64, t time.Time) {
	lock := flock.New(lockPath)
	lock.Lock()
	defer lock.Unlock()

	existing := map[string][]fileRecord{
		"default":  {},
		"fallback": {},
	}
	_ = os.MkdirAll(filepath.Dir(filePath), 0755)

	if data, err := os.ReadFile(filePath); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	existing[processor] = append(existing[processor], fileRecord{
		Processor:   processor,
		Amount:      amount,
		RequestedAt: t,
	})

	out, _ := json.Marshal(existing)
	_ = os.WriteFile(filePath, out, 0644)
}

func SummaryFromFile(processor string, from, to *time.Time) Summary {
	lock := flock.New(lockPath)
	lock.Lock()
	defer lock.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return Summary{}
	}

	records := map[string][]fileRecord{
		"default":  {},
		"fallback": {},
	}
	_ = json.Unmarshal(data, &records)

	var total Summary
	for _, rec := range records[processor] {
		if from != nil && rec.RequestedAt.Before(*from) {
			continue
		}
		if to != nil && rec.RequestedAt.After(*to) {
			continue
		}
		total.TotalRequests++
		total.TotalAmount += rec.Amount
	}

	return total
}
