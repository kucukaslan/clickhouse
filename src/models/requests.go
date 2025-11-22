package models

/**
{
"event_name": "product_view"
"channel": "web"
,
"campaign_id": "cmp_987"
,
"user_id": "user_123"
,
"timestamp": 1723475612,
"tags": ["electronics"
,
"metadata": {
"product_id": "prod-789"
,
"price": 129.99,
"currency": "TRY"
,
"referrer": "google"
,
"homepage"
,
"flash_sale"],
}
}
event_name (string): Name of the event, e.g., product_view, add_to_cart
channel (string): Source of the event, e.g., web, mobile_app, api
campaign_id (string): Identifier of the campaign associated with the event
user_id (string): ID of the user who triggered the event
timestamp (int): Unix epoch time in seconds (must not be in the future)
tags (array of strings): A list of tags associated with the event (can be used for
filtering/grouping)
metadata (object): Flexible key-value map for any additional data (product info,
referrer, currency, etc.)
*/

type EventRequest struct {
	EventName  string            `json:"event_name"`
	Channel    string            `json:"channel"`
	CampaignID string            `json:"campaign_id"`
	UserID     string            `json:"user_id"`
	Timestamp  int64             `json:"timestamp"`
	Tags       []string          `json:"tags"`
	Metadata   map[string]string `json:"metadata"`
}

/*
*
●
●
The metrics API must:
○
Support time range filtering (e.g., from, to timestamps)
○
Allow filtering by event_name (mandatory)
○
Return both:
■ Total event count
■ Unique event count (based on user_id)
At minimum, metrics should support aggregation over one additional field, such
as:
●
○
channel: group event counts by source (web, mobile, etc.)
○
or timestamp: group events daily/hourly within the given range
(Performance Note):
○
The metrics endpoint is not required to be fully real-time.
○
Howe
*/
type MetricRequest struct {
	EventName string            `json:"event_name"`
	From      int64             `json:"from"`
	To        int64             `json:"to"`
	GroupBy   string            `json:"group_by"`  // e.g., "channel" or "timestamp"
	Aggregate map[string]string `json:"aggregate"` // e.g., {"metadata.amount": "sum", "metadata.count": "avg", ...}
}
