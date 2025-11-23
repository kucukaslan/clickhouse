package domain

import "context"

type EventService interface {
	PostEvents(ctx context.Context, eventData *EventRequest) (*EventResponse, error)
	PostEventsBulk(ctx context.Context, bulkData *BulkEventRequest) (*BulkEventResponse, error)
	GetMetrics(ctx context.Context, metricRequest *MetricRequest) (*MetricResponse, error)
}

// TODO Health Service
