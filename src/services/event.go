package services

import (
	"context"
	"fmt"
	"kucukaslan/clickhouse/config"
	"kucukaslan/clickhouse/database"
	"kucukaslan/clickhouse/domain"
)

var _ domain.EventService = &eventService{}

type eventService struct {
	clickhouseDB  database.ClickHouseDB
	clickhouseCfg *config.ClickHouseConfig
	redisRepo     database.ClickHouseRedis
	batcher       *EventBatcher
}

func (e eventService) PostEvents(ctx context.Context, eventData *domain.EventRequest) (*domain.EventResponse, error) {

	// Check Redis cache for duplicate event
	isProcessed, err := e.redisRepo.IsEventProcessed(ctx, *eventData)
	if err != nil { /* do nothing or log? */
	}
	if isProcessed {
		return &domain.EventResponse{
			Success: true,
			Message: "Event already processed",
		}, nil
	}

	// Enqueue event to batcher (non-blocking)
	if err := e.batcher.Enqueue(*eventData); err != nil {
		// If buffer is full, return error (will be handled as 503 in HTTP handler)
		return &domain.EventResponse{
			Success: false,
			Message: "Event buffer is full, please try again later",
		}, err
	}

	return &domain.EventResponse{
		Success: true,
		Message: "Event posted successfully",
	}, nil
}

// get unique keys, check redis with mget, filter processed events
func (e eventService) filterProcessedEvents(events []domain.EventRequest) []domain.EventRequest {
	unprocessedEvents := make([]domain.EventRequest, 0, len(events))
	maps, err := e.redisRepo.AreEventsProcessed(context.Background(), events)
	if err != nil {
		return events
	}
	for _, event := range events {
		if processed, exists := maps[event.GetUniqueKey()]; !exists || !processed {
			unprocessedEvents = append(unprocessedEvents, event)
		}
	}
	return unprocessedEvents
}

func (e eventService) PostEventsBulk(ctx context.Context, bulkData *domain.BulkEventRequest) (*domain.BulkEventResponse, error) {
	totalCount := len(bulkData.Events)
	filteredEvents := e.filterProcessedEvents(bulkData.Events)

	if err := e.clickhouseDB.SaveEvents(ctx, filteredEvents); err != nil {
		return &domain.BulkEventResponse{
			Success:      false,
			Message:      "Failed to save bulk events: " + err.Error(),
			TotalCount:   totalCount,
			SuccessCount: 0,
			FailureCount: totalCount,
		}, err
	}

	go func() {
		err := e.redisRepo.SetMultipleEventsProcessed(ctx, filteredEvents)
		if err != nil {
			// log error
		}
	}()

	return &domain.BulkEventResponse{
		Success:      true,
		Message:      "Bulk events posted successfully",
		TotalCount:   totalCount,
		SuccessCount: totalCount,
		FailureCount: 0,
	}, nil
}

func (e eventService) GetMetrics(ctx context.Context, metricRequest *domain.MetricRequest) (*domain.MetricResponse, error) {
	metrics, err := e.clickhouseDB.GetMetrics(ctx, *metricRequest)
	if err != nil {
		return &domain.MetricResponse{
			Success: false,
			Message: "Failed to retrieve metrics: " + err.Error(),
			Metrics: nil,
		}, err
	}

	return &domain.MetricResponse{
		Success: true,
		Message: "Metrics retrieved successfully",
		Metrics: func() []domain.MetricResult {
			results := make([]domain.MetricResult, len(metrics))
			for i, m := range metrics {
				results[i] = domain.MetricResult{
					Bucket:      m.Bucket,
					TotalEvents: m.TotalEvents,
					UniqueUsers: m.UniqueUsers,
				}
			}
			return results
		}(),
	}, nil
}

// NewEventService returns a domain.EventService backed by the provided database connections.
func NewEventService(db database.ClickHouseDB, cfg *config.ClickHouseConfig, redisClient database.ClickHouseRedis) (domain.EventService, error) {
	if db.DB == nil {
		return nil, fmt.Errorf("ClickHouse database connection cannot be nil")
	}
	if redisClient.Client == nil {
		return nil, fmt.Errorf("Redis client cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("ClickHouse config cannot be nil")
	}

	// Create and start event batcher
	batcher := NewEventBatcher(
		cfg.BufferChannelCapacity,
		cfg.BatchSize,
		cfg.FlushIntervalSeconds,
		db,
		redisClient,
	)
	batcher.Start()

	srv := &eventService{
		clickhouseDB:  db,
		clickhouseCfg: cfg,
		redisRepo:     redisClient,
		batcher:       batcher,
	}
	return srv, nil
}

// Shutdown gracefully shuts down the event service and its batcher
func (e *eventService) Shutdown() error {
	if e.batcher != nil {
		return e.batcher.Shutdown()
	}
	return nil
}

// ShutdownEventService gracefully shuts down an event service if it supports shutdown
func ShutdownEventService(service domain.EventService) error {
	if srv, ok := service.(interface{ Shutdown() error }); ok {
		return srv.Shutdown()
	}
	return nil
}
