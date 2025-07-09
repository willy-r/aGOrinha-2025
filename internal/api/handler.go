package api

import (
	"fmt"
	"gorinha-2025/internal/client"
	"gorinha-2025/internal/config"
	"gorinha-2025/internal/core"
	"gorinha-2025/internal/store"
	"gorinha-2025/internal/worker"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
)

type Router struct {
	pool *worker.WorkerPool
}

func NewRouter() *Router {
	defaultURL := config.GetEnv("PAYMENT_PROCESSOR_URL_DEFAULT", "http://localhost:8001")
	fallbackURL := config.GetEnv("PAYMENT_PROCESSOR_URL_FALLBACK", "http://localhost:8002")
	workers, _ := strconv.Atoi(config.GetEnv("WORKERS", "8"))

	fmt.Println("Default Processor URL: ", defaultURL)
	fmt.Println("Fallback Processor URL:", fallbackURL)

	client := client.NewPaymentClient(defaultURL, fallbackURL)
	pool := worker.NewWorkerPool(workers, client)
	pool.Start()

	return &Router{
		pool: pool,
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

	success := r.pool.Enqueue(payment)

	if success {
		ctx.SetStatusCode(fasthttp.StatusAccepted)
		ctx.SetContentType("application/json")
		ctx.SetBodyString(`{"status":"accepted"}`)
	} else {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		ctx.SetContentType("application/json")
		ctx.SetBodyString(`{"error":"queue full"}`)
	}
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

	def := store.SummaryFromFile("default", from, to)
	fbk := store.SummaryFromFile("fallback", from, to)

	body := fmt.Sprintf(
		`{"default":{"totalRequests":%d,"totalAmount":%.2f},"fallback":{"totalRequests":%d,"totalAmount":%.2f}}`,
		def.TotalRequests, def.TotalAmount, fbk.TotalRequests, fbk.TotalAmount,
	)

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	ctx.SetBodyString(body)
}
