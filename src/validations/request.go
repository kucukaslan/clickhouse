package validations

import (
	"fmt"
	"kucukaslan/clickhouse/domain"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func ValidateEventRequest(request *domain.EventRequest) error {
	if strings.TrimSpace(request.EventName) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "event_name is required")
	}
	if strings.TrimSpace(request.Channel) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "channel is required")
	}
	if request.Timestamp <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "timestamp is required and must be a positive integer")
	}
	// TODO: add 1 second slack to account for possible clock time differences among the clients and the backend service
	if request.Timestamp > time.Now().UTC().Unix() {
		return fiber.NewError(fiber.StatusBadRequest, "timestamp cannot be in the future")
	}
	if request.UserID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "user_id is required")
	}
	if request.CampaignID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "campaign_id is required")
	}
	if request.Tags == nil {
		return fiber.NewError(fiber.StatusBadRequest, "tags is required")
	}
	for _, tag := range request.Tags {
		if strings.TrimSpace(tag) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "tags cannot be empty")
		}
	}
	if request.Metadata == nil {
		return fiber.NewError(fiber.StatusBadRequest, "metadata is required")
	}
	for key, _ := range request.Metadata {
		if strings.TrimSpace(key) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "metadata keys cannot be empty")
		}
	}
	return nil
}

func ValidateMetricRequest(request *domain.MetricRequest) error {
	if request.From != nil {
		// From timestamp must be a positive and not in the future
		if *request.From <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "from must be a positive integer")
		}
		if *request.From > time.Now().UTC().Unix() {
			return fiber.NewError(fiber.StatusBadRequest, "from cannot be in the future")
		}
	}
	if request.To != nil {
		// To timestamp must be a positive and not in the future
		if *request.To <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "to must be a positive integer")
		}
		if *request.To > time.Now().UTC().Unix() {
			return fiber.NewError(fiber.StatusBadRequest, "to cannot be in the future")
		}
	}
	if request.From != nil && request.To != nil {
		if *request.From > *request.To {
			return fiber.NewError(fiber.StatusBadRequest, "from cannot be greater than to")
		}
	}

	if request.GroupBy != nil {
		if strings.TrimSpace(*request.GroupBy) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "group_by cannot be empty if provided")
		}
	}

	if request.EventName != nil {
		if strings.TrimSpace(*request.EventName) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "event_name cannot be empty if provided")
		}
	}

	return nil
}

const (
	// MaxBulkEventCount is the maximum number of events allowed in a single bulk request
	MaxBulkEventCount = 10000
)

// ValidateBulkEventRequest validates a bulk event request
// It checks batch size limits and validates each individual event
// Returns an error if any validation fails (all-or-nothing approach)
func ValidateBulkEventRequest(request *domain.BulkEventRequest) error {
	if request == nil {
		return fiber.NewError(fiber.StatusBadRequest, "bulk event request is required")
	}
	if request.Events == nil {
		return fiber.NewError(fiber.StatusBadRequest, "events array is required")
	}
	if len(request.Events) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "events array cannot be empty")
	}
	if len(request.Events) > MaxBulkEventCount {
		return fiber.NewError(fiber.StatusBadRequest, 
			"events array exceeds maximum allowed size")
	}

	// Validate each event in the batch
	for i, event := range request.Events {
		if err := ValidateEventRequest(&event); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, 
				fmt.Sprintf("validation failed for event at index %d: %v", i, err))
		}
	}

	return nil
}
