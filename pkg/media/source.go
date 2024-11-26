package media

import (
	"context"
	"time"
)

// Source represents a media source that can be scanned for tracks
type Source interface {
	// Type returns the source type identifier
	Type() string
	// Name returns a human-readable name for the source
	Name() string
	// Scan scans the source for tracks and returns them through the channel
	// The optional onFile callback is called for each file encountered, even if it's not a track
	Scan(ctx context.Context, tracks chan<- Track, onFile func(path string)) error
}

// Track represents a media track with its metadata
type Track struct {
	ID          string
	SourceID    string
	SourceType  string
	Path        string
	Title       string
	Artist      string
	Album       string
	Duration    time.Duration
	LastScanned time.Time
}

// SourceConfig represents the configuration for a media source
type SourceConfig struct {
	ID          string
	Type        string
	Name        string
	Config      map[string]string
	LastScanned time.Time
}
