package store

import (
	"sync"
	"time"
)

type Summary struct {
	TotalRequests int
	TotalAmount   float64
}

type MemoryStore struct {
	mu      sync.RWMutex
	records map[string][]PaymentRecord // key: default or "fallback"
}

type PaymentRecord struct {
	Amount float64
	Time   time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		records: map[string][]PaymentRecord{
			"default":  {},
			"fallback": {},
		},
	}
}

func (m *MemoryStore) Add(processor string, amount float64, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records[processor] = append(m.records[processor], PaymentRecord{Amount: amount, Time: t})
}

func (m *MemoryStore) Summary(processor string, from, to *time.Time) Summary {
	m.mu.Lock()
	defer m.mu.Unlock()

	var total Summary
	for _, rec := range m.records[processor] {
		if from != nil && rec.Time.Before(*from) {
			continue
		}
		if to != nil && rec.Time.After(*to) {
			continue
		}
		total.TotalRequests++
		total.TotalAmount += rec.Amount
	}

	return total
}
