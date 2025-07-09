package worker

import (
	"gorinha-2025/internal/client"
	"gorinha-2025/internal/config"
	"gorinha-2025/internal/core"
	"gorinha-2025/internal/store"
	"log"
	"strconv"
	"strings"
	"time"
)

type WorkerPool struct {
	queue   chan *core.PaymentRequest
	workers int
	client  *client.PaymentClient
}

func NewWorkerPool(size int, client *client.PaymentClient) *WorkerPool {
	queueSize, _ := strconv.Atoi(config.GetEnv("QUEUE_SIZE", "2000"))

	return &WorkerPool{
		queue:   make(chan *core.PaymentRequest, queueSize), // initial buffer
		workers: size,
		client:  client,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		go wp.worker()
	}
}

func (wp *WorkerPool) Enqueue(p *core.PaymentRequest) bool {
	select {
	case wp.queue <- p:
		return true
	default:
		log.Println("Queue full, payment discarded")
		return false
	}
}

func (wp *WorkerPool) worker() {
	for p := range wp.queue {
		t, _ := time.Parse(time.RFC3339Nano, p.RequestedAt)

		err := wp.client.SendToDefault(p)
		if err == nil {
			store.AddPaymentToFile("default", p.Amount, t)
			continue
		}

		if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "422") {
			continue
		}

		err = wp.client.SendToFallback(p)
		if err == nil {
			store.AddPaymentToFile("fallback", p.Amount, t)
		} else {
			log.Printf("Failed to process payment: %v", err)
		}
	}
}
