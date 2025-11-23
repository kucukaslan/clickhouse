package database

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"kucukaslan/clickhouse/domain"
	"log"
	"time"

	"kucukaslan/clickhouse/config"

	"github.com/uptrace/go-clickhouse/ch"
)

var clickHouseDB *ch.DB

// InitClickHouse initializes the ClickHouse database connection
func InitClickHouse(cfg *config.ClickHouseConfig) error {
	dsn := cfg.GetClickHouseDSN()

	// Connect without TLS since ClickHouse native protocol doesn't use TLS by default
	db := ch.Connect(
		ch.WithDSN(dsn),
		ch.WithInsecure(true), // Disable TLS for native protocol
	)

	// Test the connection
	ctx := context.Background()

	// Initialize events table
	if err := InitEventsTable(ctx, db); err != nil {
		return fmt.Errorf("failed to initialize events table: %w", err)
	}

	clickHouseDB = db
	log.Println("ClickHouse connection established successfully")

	return nil
}

// CloseClickHouse closes the ClickHouse database connection
func CloseClickHouse() error {
	if clickHouseDB != nil {
		if err := clickHouseDB.Close(); err != nil {
			return fmt.Errorf("failed to close ClickHouse connection: %w", err)
		}
		log.Println("ClickHouse connection closed")
	}
	return nil
}

// InitEventsTable creates the events table if it doesn't exist
func InitEventsTable(ctx context.Context, db *ch.DB) error {
	_, err := db.NewCreateTable().
		Model((*Event)(nil)).
		Engine("ReplacingMergeTree(ingested_at)").
		Order("timestamp, event_name, channel, user_id").
		IfNotExists().
		Exec(ctx)

	return err
}

// ClickHouseHealthCheck verifies that the ClickHouse connection is alive
func ClickHouseHealthCheck(ctx context.Context) error {
	if clickHouseDB == nil {
		return fmt.Errorf("ClickHouse connection is not initialized")
	}
	return clickHouseDB.Ping(ctx)
}

// GetClickHouseDB returns the ClickHouse database instance
func GetClickHouseDB() ClickHouseDB {
	return ClickHouseDB{clickHouseDB}
}

// Event represents the events table structure for ClickHouse ORM
type Event struct {
	ch.CHModel `ch:"table:events,partition:toYYYYMMDD(timestamp)"`
	EventName  string    `ch:"event_name,lc"`
	Channel    string    `ch:"channel,lc"`
	CampaignID string    `ch:"campaign_id"`
	UserID     string    `ch:"user_id"`
	Timestamp  time.Time `ch:"timestamp"`
	Tags       []string  `ch:"tags,array"`
	Metadata   string    `ch:"metadata,type:String"`

	IngestedAt time.Time `ch:"ingested_at,default:now()"`
}

// EventColumnar:  events in columnar format for batch inserts
type EventColumnar struct {
	ch.CHModel `ch:"table:events,partition:toYYYYMMDD(timestamp),columnar"`
	EventName  []string    `ch:"event_name,lc"`
	Channel    []string    `ch:"channel,lc"`
	CampaignID []string    `ch:"campaign_id"`
	UserID     []string    `ch:"user_id"`
	Timestamp  []time.Time `ch:"timestamp"`
	Tags       [][]string  `ch:"tags,array"`
	Metadata   []string    `ch:"metadata,type:String"`

	IngestedAt []time.Time `ch:"ingested_at,default:now()"`
}

// SaveEvent saves an event to ClickHouse using async insert for high throughput
// Async insert settings are configured at the connection level via DSN parameters
func (c ClickHouseDB) SaveEvent(ctx context.Context, request domain.EventRequest) error {
	if c.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	event, err := mapEventRequestToEvent(request)
	if err != nil {
		return err
	}

	_, err = c.DB.NewInsert().
		Model(event).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

// SaveEvents saves multiple events to ClickHouse using native columnar insert format
// This method uses ClickHouse's columnar insert which is significantly faster than row-based inserts
// Data is sent column-by-column as arrays, optimizing for ClickHouse's columnar storage engine
func (c ClickHouseDB) SaveEvents(ctx context.Context, requests []domain.EventRequest) error {
	if c.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	if len(requests) == 0 {
		return fmt.Errorf("no events to insert")
	}

	batchSize := len(requests)
	now := time.Now()

	eventNames := make([]string, 0, batchSize)
	channels := make([]string, 0, batchSize)
	campaignIDs := make([]string, 0, batchSize)
	userIDs := make([]string, 0, batchSize)
	timestamps := make([]time.Time, 0, batchSize)
	tags := make([][]string, 0, batchSize)
	metadata := make([]string, 0, batchSize)
	ingestedAt := make([]time.Time, 0, batchSize)

	// Extract columns from requests
	for _, request := range requests {
		// Serialize metadata to JSON string
		metadataJSON := ""
		if request.Metadata != nil && len(request.Metadata) > 0 {
			metadataBytes, err := json.Marshal(request.Metadata)
			if err != nil {
				return fmt.Errorf("failed to serialize metadata: %w", err)
			}
			metadataJSON = string(metadataBytes)
		}

		// Convert Unix timestamp to DateTime
		eventTime := time.Unix(request.Timestamp, 0)

		// Append to columnar arrays
		eventNames = append(eventNames, request.EventName)
		channels = append(channels, request.Channel)
		campaignIDs = append(campaignIDs, request.CampaignID)
		userIDs = append(userIDs, request.UserID)
		timestamps = append(timestamps, eventTime)
		tags = append(tags, request.Tags)
		metadata = append(metadata, metadataJSON)
		ingestedAt = append(ingestedAt, now)
	}

	// Create columnar model
	columnarModel := &EventColumnar{
		EventName:  eventNames,
		Channel:    channels,
		CampaignID: campaignIDs,
		UserID:     userIDs,
		Timestamp:  timestamps,
		Tags:       tags,
		Metadata:   metadata,
		IngestedAt: ingestedAt,
	}

	_, err := c.DB.NewInsert().
		Model(columnarModel).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to columnar insert events: %w", err)
	}

	return nil
}
func mapEventRequestToEvent(request domain.EventRequest) (*Event, error) {
	// Serialize metadata to JSON string
	metadataJSON := ""
	if request.Metadata != nil && len(request.Metadata) > 0 {
		metadataBytes, err := json.Marshal(request.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	// Convert Unix timestamp to DateTime
	eventTime := time.Unix(request.Timestamp, 0)

	event := &Event{
		EventName:  request.EventName,
		Channel:    request.Channel,
		CampaignID: request.CampaignID,
		UserID:     request.UserID,
		Timestamp:  eventTime,
		Tags:       request.Tags,
		Metadata:   metadataJSON,
	}
	return event, nil
}

type MetricResult struct {
	// The "Bucket" holds the group name (e.g., "2024-08-25 10:00:00" or "mobile")
	Bucket      string `ch:"bucket"`
	TotalEvents uint64 `ch:"total_events"`
	UniqueUsers uint64 `ch:"unique_users"`
}

// GetMetrics retrieves aggregated metrics from events table
func (c ClickHouseDB) GetMetrics(ctx context.Context, request domain.MetricRequest) ([]MetricResult, error) {
	var results []MetricResult

	// 1. Determine the Grouping Logic safely
	// Prevents SQL injection by validating the input against an allowlist.
	var groupExpr string
	if request.GroupBy != nil {
		switch *request.GroupBy {
		case "hour":
			groupExpr = "toString(toStartOfHour(timestamp))"
		case "day":
			groupExpr = "toString(toStartOfDay(timestamp))"
		case "week":
			groupExpr = "toString(toStartOfWeek(timestamp))"
		case "month":
			groupExpr = "toString(toStartOfMonth(timestamp))"
		case "year":
			groupExpr = "toString(toStartOfYear(timestamp))"
		case "channel":
			groupExpr = "channel"
		case "campaign_id":
			groupExpr = "campaign_id"
		case "user_id":
			groupExpr = "user_id"
		case "event_name":
			groupExpr = "event_name"
		default:
			// Default fallback (e.g., if they didn't provide a valid group)
		}
	}

	query := c.NewSelect().
		// Explicitly use TableExpr to add 'FINAL'.
		// This forces ClickHouse to deduplicate rows before counting.
		TableExpr("events FINAL")

	if groupExpr != "" {
		query = query.ColumnExpr("? AS bucket", ch.Safe(groupExpr))
	} else {
		query = query.ColumnExpr("'total' AS bucket")
	}
	query = query.
		ColumnExpr("count() AS total_events").
		ColumnExpr("uniqExact(user_id) AS unique_users")

	if request.EventName != nil && *request.EventName != "" {
		query = query.Where("event_name = ?", *request.EventName)
	}
	if request.From != nil {
		fromTime := time.Unix(*request.From, 0)
		query = query.Where("timestamp >= ?", fromTime)
	}
	if request.To != nil {
		toTime := time.Unix(*request.To, 0)
		query = query.Where("timestamp <= ?", toTime)
	}
	if groupExpr != "" {
		query = query.GroupExpr(groupExpr)
		query = query.OrderExpr("bucket ASC")
	}

	err := query.Scan(ctx, &results)
	if err != nil {
		return nil, err
	}

	return results, err
}

type ClickHouseDB struct {
	*ch.DB
}
