package domain

import "strconv"

// EventRequest represents an event to be tracked
type EventRequest struct {
	EventName  string         `json:"event_name" example:"purchase"`
	Channel    string         `json:"channel" example:"web"`
	CampaignID string         `json:"campaign_id" example:"summer_sale_2025"`
	UserID     string         `json:"user_id" example:"user123"`
	Timestamp  int64          `json:"timestamp" example:"1732233600" minimum:"0"`
	Tags       []string       `json:"tags" example:"mobile,premium"`
	Metadata   map[string]any `json:"metadata" swaggertype:"object"`
}

// `event_name, user_id, timestamp, channel` pair as a unique identifier
func (e EventRequest) GetUniqueKey() string {
	return e.EventName + "|" + e.UserID + "|" + strconv.FormatInt(e.Timestamp, 10) + "|" + e.Channel
}

// MetricRequest represents a query for aggregated metrics
type MetricRequest struct {
	EventName *string `json:"event_name" example:"purchase"`
	From      *int64  `json:"from" example:"1732147200"`
	To        *int64  `json:"to" example:"1732233600"`
	GroupBy   *string `json:"group_by" example:"channel"` // e.g., "channel" or "timestamp"
}

// BulkEventRequest represents a batch of events to be tracked
type BulkEventRequest struct {
	Events []EventRequest `json:"events"`
}
