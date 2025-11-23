package buildinfo

import (
	"os"
	"runtime"
	"time"
)

// Build information variables set via ldflags during compilation
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// startTime tracks when the application started
var startTime = time.Now()

// Info contains build and runtime information
type Info struct {
	Version   string        `json:"version" example:"v1.0.0"`
	Commit    string        `json:"commit" example:"abc123def456"`
	BuildDate string        `json:"buildDate" example:"2025-11-22T10:00:00Z"`
	GoVersion string        `json:"goVersion" example:"go1.25.4"`
	Hostname  string        `json:"hostname" example:"app-server-01"`
	Uptime    time.Duration `json:"uptime" swaggertype:"integer" example:"3600000000000"`
}

// GetInfo returns complete build and runtime information
func GetInfo() Info {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Hostname:  hostname,
		Uptime:    time.Since(startTime),
	}
}

// SetStartTime allows overriding the start time (useful for tracking from main)
func SetStartTime(t time.Time) {
	startTime = t
}
