package services

import (
	"context"
	"errors"
	"kucukaslan/clickhouse/database"
	"kucukaslan/clickhouse/domain"
	"log"
	"sync"
	"time"
)

var (
	// ErrBufferFull is returned when the event buffer channel is full
	ErrBufferFull = errors.New("event buffer is full")
)

// EventBatcher batches events and flushes them to ClickHouse
type EventBatcher struct {
	eventChan        chan domain.EventRequest
	batchSize        int
	flushInterval    time.Duration
	clickhouseDB     database.ClickHouseDB
	redisRepo        database.ClickHouseRedis
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	mu               sync.Mutex
	isRunning        bool
	currentBatch     []domain.EventRequest
	lastFlushTime    time.Time
}

// NewEventBatcher creates a new EventBatcher instance
func NewEventBatcher(
	capacity int,
	batchSize int,
	flushIntervalSeconds int,
	clickhouseDB database.ClickHouseDB,
	redisRepo database.ClickHouseRedis,
) *EventBatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventBatcher{
		eventChan:     make(chan domain.EventRequest, capacity),
		batchSize:     batchSize,
		flushInterval: time.Duration(flushIntervalSeconds) * time.Second,
		clickhouseDB:  clickhouseDB,
		redisRepo:     redisRepo,
		ctx:           ctx,
		cancel:        cancel,
		currentBatch:  make([]domain.EventRequest, 0, batchSize),
		lastFlushTime: time.Now(),
	}
}

// Start launches the background worker goroutine that processes events
func (b *EventBatcher) Start() {
	b.mu.Lock()
	if b.isRunning {
		b.mu.Unlock()
		return
	}
	b.isRunning = true
	b.mu.Unlock()

	b.wg.Add(1)
	go b.worker()
	log.Println("EventBatcher started")
}

// Enqueue adds an event to the buffer channel (non-blocking)
// Returns ErrBufferFull if the channel is full
func (b *EventBatcher) Enqueue(event domain.EventRequest) error {
	select {
	case b.eventChan <- event:
		return nil
	default:
		return ErrBufferFull
	}
}

// worker is the background goroutine that collects events and flushes them
func (b *EventBatcher) worker() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			// Flush remaining events before shutting down
			b.flushRemaining()
			return

		case event := <-b.eventChan:
			b.mu.Lock()
			b.currentBatch = append(b.currentBatch, event)
			shouldFlush := len(b.currentBatch) >= b.batchSize
			b.mu.Unlock()

			if shouldFlush {
				b.flushBatch()
			}

		case <-ticker.C:
			// Time-based flush
			b.mu.Lock()
			hasEvents := len(b.currentBatch) > 0
			b.mu.Unlock()

			if hasEvents {
				b.flushBatch()
			}
		}
	}
}

// flushBatch flushes the current batch to ClickHouse
func (b *EventBatcher) flushBatch() {
	b.mu.Lock()
	if len(b.currentBatch) == 0 {
		b.mu.Unlock()
		return
	}

	// Copy batch and clear current batch
	batch := make([]domain.EventRequest, len(b.currentBatch))
	copy(batch, b.currentBatch)
	b.currentBatch = b.currentBatch[:0]
	b.mu.Unlock()

	// Filter processed events using Redis
	unprocessedEvents := b.filterProcessedEvents(batch)

	if len(unprocessedEvents) == 0 {
		log.Printf("EventBatcher: All %d events in batch were already processed", len(batch))
		return
	}

	// Save to ClickHouse
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := b.clickhouseDB.SaveEvents(ctx, unprocessedEvents); err != nil {
		log.Printf("EventBatcher: Failed to flush batch of %d events: %v", len(unprocessedEvents), err)
		return
	}

	log.Printf("EventBatcher: Successfully flushed batch of %d events (filtered from %d)", len(unprocessedEvents), len(batch))

	// Mark events as processed in Redis (async)
	go func() {
		if err := b.redisRepo.SetMultipleEventsProcessed(context.Background(), unprocessedEvents); err != nil {
			log.Printf("EventBatcher: Failed to mark events as processed in Redis: %v", err)
		}
	}()
}

// flushRemaining flushes any remaining events in the buffer during shutdown
func (b *EventBatcher) flushRemaining() {
	b.mu.Lock()
	remaining := len(b.currentBatch)
	b.mu.Unlock()

	if remaining > 0 {
		log.Printf("EventBatcher: Flushing %d remaining events during shutdown", remaining)
		b.flushBatch()
	}

	// Drain any remaining events from the channel
	drained := 0
	for {
		select {
		case event := <-b.eventChan:
			b.mu.Lock()
			b.currentBatch = append(b.currentBatch, event)
			b.mu.Unlock()
			drained++
		default:
			if drained > 0 {
				log.Printf("EventBatcher: Drained %d events from channel during shutdown", drained)
				b.flushBatch()
			}
			return
		}
	}
}

// filterProcessedEvents filters out events that have already been processed
func (b *EventBatcher) filterProcessedEvents(events []domain.EventRequest) []domain.EventRequest {
	unprocessedEvents := make([]domain.EventRequest, 0, len(events))
	maps, err := b.redisRepo.AreEventsProcessed(context.Background(), events)
	if err != nil {
		// If Redis check fails, assume all events are unprocessed
		log.Printf("EventBatcher: Redis check failed, assuming all events are unprocessed: %v", err)
		return events
	}

	for _, event := range events {
		if processed, exists := maps[event.GetUniqueKey()]; !exists || !processed {
			unprocessedEvents = append(unprocessedEvents, event)
		}
	}
	return unprocessedEvents
}

// Shutdown gracefully shuts down the batcher, flushing remaining events
func (b *EventBatcher) Shutdown() error {
	b.mu.Lock()
	if !b.isRunning {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	log.Println("EventBatcher: Initiating graceful shutdown...")
	b.cancel()
	b.wg.Wait()
	log.Println("EventBatcher: Shutdown complete")
	return nil
}

// GetBufferSize returns the current number of events in the buffer channel
func (b *EventBatcher) GetBufferSize() int {
	return len(b.eventChan)
}

// GetBatchSize returns the current number of events in the pending batch
func (b *EventBatcher) GetBatchSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.currentBatch)
}

