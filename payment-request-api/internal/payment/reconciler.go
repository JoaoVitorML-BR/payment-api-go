// payment-request-api/internal/payment/reconciler.go
package payment

import (
	"context"
	"log"
	"sync"
	"time"
)

type ReconcilerWorker struct {
	service    *PaymentService
	interval   time.Duration
	minAge     time.Duration
	batchLimit int32
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

func NewReconcilerWorker(service *PaymentService, interval, minAge time.Duration, batchLimit int32) *ReconcilerWorker {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if minAge <= 0 {
		minAge = 2 * time.Minute
	}
	if batchLimit <= 0 {
		batchLimit = 50
	}

	return &ReconcilerWorker{
		service:    service,
		interval:   interval,
		minAge:     minAge,
		batchLimit: batchLimit,
		stopChan:   make(chan struct{}),
	}
}

// Start launches the background reconciliation goroutine.
func (w *ReconcilerWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		log.Printf("[RECONCILIATION] Reconciliation worker started (interval: %v, minAge: %v)", w.interval, w.minAge)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[RECONCILIATION] Worker stopping due to context cancellation")
				return
			case <-w.stopChan:
				log.Println("[RECONCILIATION] Worker stopping due to stop signal")
				return
			case <-ticker.C:
				count, err := w.service.ReconcilePendingPayments(ctx, w.minAge, w.batchLimit)
				if err != nil {
					log.Printf("[RECONCILIATION] Error during reconciliation cycle: %v", err)
				} else if count > 0 {
					log.Printf("[RECONCILIATION] Reconciliation cycle finished: %d payments reconciled", count)
				}
			}
		}
	}()
}

// Stop signals the background worker to stop and waits for it to terminate.
func (w *ReconcilerWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	log.Println("[RECONCILIATION] Worker stopped cleanly")
}
