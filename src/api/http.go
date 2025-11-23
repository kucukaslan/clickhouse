package api

import (
	"errors"
	"kucukaslan/clickhouse/domain"
	"kucukaslan/clickhouse/services"
	"kucukaslan/clickhouse/validations"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

var _ EventHandler = &eventHandler{nil}

type eventHandler struct {
	eventService domain.EventService
}

// PostEvent handles posting events
// @Summary Post event data
// @Description Submit event data for tracking and analytics
// @Tags Events
// @Accept json
// @Produce json
// @Param event body domain.EventRequest true "Event data"
// @Success 200 {object} domain.EventResponse "Event posted successfully"
// @Failure 400 {object} domain.EventResponse "Invalid request"
// @Failure 503 {object} domain.EventResponse "Service unavailable (buffer full)"
// @Failure 500 {object} domain.EventResponse "Internal server error"
// @Router /events [post]
func (e eventHandler) PostEvent(ctx *fiber.Ctx) error {
	// Parse request body
	var req domain.EventRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(domain.EventResponse{
			Success: false,
			Message: "Invalid request body: " + err.Error(),
		})
	}

	// Validate request
	if err := validations.ValidateEventRequest(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(domain.EventResponse{
			Success: false,
			Message: "Validation failed: " + err.Error(),
		})
	}

	resp, err := e.eventService.PostEvents(ctx.Context(), &req)
	if err != nil {
		// Check if buffer is full and return 503 Service Unavailable
		if errors.Is(err, services.ErrBufferFull) {
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(domain.EventResponse{
				Success: false,
				Message: "Service temporarily unavailable, please try again later",
			})
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(domain.EventResponse{
			Success: false,
			Message: "Internal server error: " + err.Error(),
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(resp)
}

// GetMetrics retrieves aggregated metrics
// @Summary GET aggregated metrics
// @Description Query aggregated event metrics with filtering and grouping
// @Tags Metrics
// @Produce json
// @Param event_name query string false "Event name filter"
// @Param from query int false "Start timestamp (Unix seconds)"
// @Param to query int false "End timestamp (Unix seconds)"
// @Param group_by query string false "Group by field (hour, day, week, month, year, channel, campaign_id, user_id, event_name)"
// @Success 200 {object} domain.MetricResponse "Metrics retrieved successfully"
// @Failure 400 {object} domain.MetricResponse "Invalid request"
// @Failure 500 {object} domain.MetricResponse "Internal server error"
// @Router /metrics [get]
func (e eventHandler) GetMetrics(ctx *fiber.Ctx) error {
	// Parse query parameters
	var req domain.MetricRequest

	// Parse event_name
	if eventName := ctx.Query("event_name"); eventName != "" {
		req.EventName = &eventName
	}

	// Parse from timestamp
	if fromStr := ctx.Query("from"); fromStr != "" {
		from, err := strconv.ParseInt(fromStr, 10, 64)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(domain.MetricResponse{
				Success: false,
				Message: "Invalid 'from' parameter: " + err.Error(),
				Metrics: nil,
			})
		}
		req.From = &from
	}

	// Parse to timestamp
	if toStr := ctx.Query("to"); toStr != "" {
		to, err := strconv.ParseInt(toStr, 10, 64)
		if err != nil {
			return ctx.Status(fiber.StatusBadRequest).JSON(domain.MetricResponse{
				Success: false,
				Message: "Invalid 'to' parameter: " + err.Error(),
				Metrics: nil,
			})
		}
		req.To = &to
	}

	// Parse group_by
	if groupBy := ctx.Query("group_by"); groupBy != "" {
		req.GroupBy = &groupBy
	}

	// Validate request
	if err := validations.ValidateMetricRequest(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(domain.MetricResponse{
			Success: false,
			Metrics: nil,
			Message: "Validation failed: " + err.Error(),
		})
	}
	resp, err := e.eventService.GetMetrics(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(domain.MetricResponse{
			Success: false,
			Message: "Internal server error: " + err.Error(),
			Metrics: resp.Metrics,
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(resp)

}

// PostEventsBulk handles posting multiple events in bulk
// @Summary Post bulk event data
// @Description Submit multiple events in a single request for high-throughput ingestion. Uses columnar batch inserts for optimal performance.
// @Tags Events
// @Accept json
// @Produce json
// @Param events body domain.BulkEventRequest true "Array of event data"
// @Success 200 {object} domain.BulkEventResponse "Bulk events posted successfully"
// @Failure 400 {object} domain.BulkEventResponse "Invalid request"
// @Failure 500 {object} domain.BulkEventResponse "Internal server error"
// @Router /events/bulk [post]
func (e eventHandler) PostEventsBulk(ctx *fiber.Ctx) error {
	// Parse request body
	var req domain.BulkEventRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(domain.BulkEventResponse{
			Success:      false,
			Message:      "Invalid request body: " + err.Error(),
			TotalCount:   0,
			SuccessCount: 0,
			FailureCount: 0,
		})
	}

	// Validate request
	if err := validations.ValidateBulkEventRequest(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(domain.BulkEventResponse{
			Success:      false,
			Message:      "Validation failed: " + err.Error(),
			TotalCount:   len(req.Events),
			SuccessCount: 0,
			FailureCount: len(req.Events),
		})
	}

	resp, err := e.eventService.PostEventsBulk(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(domain.BulkEventResponse{
			Success:      false,
			Message:      "Internal server error: " + err.Error(),
			TotalCount:   resp.TotalCount,
			SuccessCount: resp.SuccessCount,
			FailureCount: resp.FailureCount,
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(resp)
}

func NewEventHandler(eventService domain.EventService) EventHandler {
	return &eventHandler{eventService: eventService}
}
