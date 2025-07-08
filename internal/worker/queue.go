package worker

import (
	"gorinha-2025/internal/client"
	"gorinha-2025/internal/core"
	"gorinha-2025/internal/store"
	"log"
	"time"
)

type WorkerPool struct {
	queue   chan *core.PaymentRequest
	workers int
	client  *client.PaymentClient
	store   *store.MemoryStore
}

func NewWorkerPool(size int, client *client.PaymentClient, store *store.MemoryStore) *WorkerPool {
	return &WorkerPool{
		queue:   make(chan *core.PaymentRequest, 10000), // initial buffer
		workers: size,
		client:  client,
		store:   store,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		go wp.worker()
	}
}

func (wp *WorkerPool) Enqueue(p *core.PaymentRequest) {
	select {
	case wp.queue <- p:
	default:
		log.Println("Queue full, payment discarded")
	}
}

func (wp *WorkerPool) worker() {
	for p := range wp.queue {
		processor := "default"
		err := wp.client.SendToDefault(p)
		if err != nil {
			log.Printf("Attempting fallback to %s: %v", p.CorrelationID, err)
			err = wp.client.SendToFallback(p)
			if err != nil {
				log.Printf("GENERAL FAIL PROCESSING: %s | err=%v", p.CorrelationID, err)
				continue
			}
			processor = "fallback"
			log.Printf("fallback SUCCESS: %s | amount=%.2f", p.CorrelationID, p.Amount)
		} else {
			log.Printf("default SUCCESS: %s | amount=%.2f", p.CorrelationID, p.Amount)
		}

		t, _ := time.Parse(time.RFC3339Nano, p.RequestedAt)
		wp.store.Add(processor, p.Amount, t)
	}
}
