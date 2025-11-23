package domain

import (
	"kucukaslan/clickhouse/buildinfo"
	"time"
)

// HealthResponse represents the health status of the service
type HealthResponse struct {
	Status    string              `json:"status" example:"healthy"`
	Timestamp time.Time           `json:"timestamp" example:"2025-11-22T10:00:00Z"`
	BuildInfo buildinfo.Info      `json:"buildInfo"`
	Services  ServiceHealthStatus `json:"services"`
}

// ServiceHealthStatus represents the health status of dependent services
type ServiceHealthStatus struct {
	ClickHouse ServiceStatus `json:"clickhouse"`
	Redis      ServiceStatus `json:"redis"`
}

// ServiceStatus represents the status of a single service
type ServiceStatus struct {
	Status  string `json:"status" example:"healthy"`
	Message string `json:"message,omitempty" example:""`
}

// EventResponse represents the response after posting an event
type EventResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Event posted successfully"`
}

// MetricResponse represents aggregated metrics data
type MetricResponse struct {
	Success bool           `json:"success" example:"true"`
	Message string         `json:"message" example:"Metrics retrieved successfully"`
	Metrics []MetricResult `json:"metrics"`
}

type MetricResult struct {
	// The "Bucket" holds the group name (e.g., "2024-08-25 10:00:00" or "mobile")
	Bucket      string `json:"bucket"`
	TotalEvents uint64 `json:"total_events"`
	UniqueUsers uint64 `json:"unique_users"`
}

// BulkEventResponse represents the response after posting bulk events
type BulkEventResponse struct {
	Success     bool   `json:"success" example:"true"`
	Message     string `json:"message" example:"Bulk events posted successfully"`
	TotalCount  int    `json:"total_count" example:"100"`
	SuccessCount int   `json:"success_count" example:"100"`
	FailureCount int   `json:"failure_count" example:"0"`
}
