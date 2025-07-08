package worker

import (
	"gorinha-2025/internal/client"
	"gorinha-2025/internal/core"
	"gorinha-2025/internal/store"
	"log"
	"strings"
	"time"
)

type WorkerPool struct {
	queue   chan *core.PaymentRequest
	workers int
	client  *client.PaymentClient
}

func NewWorkerPool(size int, client *client.PaymentClient) *WorkerPool {
	return &WorkerPool{
		queue:   make(chan *core.PaymentRequest, 10000), // initial buffer
		workers: size,
		client:  client,
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
		t, _ := time.Parse(time.RFC3339Nano, p.RequestedAt)

		err := wp.client.SendToDefault(p)

		if err != nil {
			if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "422") {
				log.Printf("Payment skipped (client reject or duplicate): %s", p.CorrelationID)
				continue
			}

			log.Printf("Attempting fallback for %s: %v", p.CorrelationID, err)
			err = wp.client.SendToFallback(p)
			if err != nil {
				log.Printf("GENERAL FAIL PROCESSING: %s | err=%v", p.CorrelationID, err)
				continue
			}

			store.AddPaymentToFile("fallback", p.Amount, t)
			log.Printf("fallback SUCCESS: %s | amount=%.2f", p.CorrelationID, p.Amount)
			continue
		}

		store.AddPaymentToFile("default", p.Amount, t)
		log.Printf("default SUCCESS: %s | amount=%.2f", p.CorrelationID, p.Amount)
	}
}
