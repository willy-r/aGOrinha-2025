package api

import (
	"fmt"
	"gorinha-2025/internal/client"
	"gorinha-2025/internal/config"
	"gorinha-2025/internal/core"
	"gorinha-2025/internal/store"
	"gorinha-2025/internal/worker"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

type Router struct {
	store *store.MemoryStore
	pool  *worker.WorkerPool
}

func NewRouter() *Router {
	storeInstance := store.NewMemoryStore()

	defaultURL := config.GetEnv("PAYMENT_PROCESSOR_URL_DEFAULT", "http://localhost:8001")
	fallbackURL := config.GetEnv("PAYMENT_PROCESSOR_URL_FALLBACK", "http://localhost:8002")

	fmt.Println("Default Processor URL: ", defaultURL)
	fmt.Println("Fallback Processor URL:", fallbackURL)

	client := client.NewPaymentClient(defaultURL, fallbackURL)
	pool := worker.NewWorkerPool(8, client, storeInstance)
	pool.Start()

	return &Router{
		store: store.NewMemoryStore(),
		pool:  pool,
	}
}

func (r *Router) HandleRequest(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/payments":
		if ctx.IsPost() {
			r.handlePostPayments(ctx)
			return
		}
	case "/payments-summary":
		if ctx.IsGet() {
			r.handleGetPaymentsSummary(ctx)
			return
		}
	}

	ctx.SetStatusCode(fasthttp.StatusNotFound)
	ctx.SetBodyString("404 - Not Found")
}

func (r *Router) handlePostPayments(ctx *fasthttp.RequestCtx) {
	body := ctx.PostBody()
	payment, err := core.ParseAndValidatePayment(body)

	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(err.Error())
		return
	}

	log.Printf("Valid payment received, sending to queue: %v", payment)

	r.pool.Enqueue(payment)
	ctx.SetStatusCode(fasthttp.StatusAccepted)
}

func (r *Router) handleGetPaymentsSummary(ctx *fasthttp.RequestCtx) {
	fromStr := string(ctx.QueryArgs().Peek("from"))
	toStr := string(ctx.QueryArgs().Peek("to"))

	var from, to *time.Time

	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, fromStr); err == nil {
			from = &t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, toStr); err == nil {
			to = &t
		}
	}

	def := r.store.Summary("default", from, to)
	fbk := r.store.Summary("fallback", from, to)

	body := fmt.Sprintf(
		`{"default":{"totalRequests":%d,"totalAmount":%.2f},"fallback":{"totalRequests":%d,"totalAmount":%.2f}}`,
		def.TotalRequests, def.TotalAmount, fbk.TotalRequests, fbk.TotalAmount,
	)

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(body)
}
