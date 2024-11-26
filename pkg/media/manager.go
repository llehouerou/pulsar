package media

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SourceManager handles media source registration and scanning
type SourceManager struct {
	db interface {
		SaveSource(source *SourceConfig) error
		GetSources() ([]SourceConfig, error)
		SaveTrack(track *Track) error
		GetTracks(sourceID string) ([]Track, error)
	}
	sources         map[string]Source
	sourceFactories map[string]SourceFactory
	scanProgress    *ScanProgress
	mu              sync.RWMutex
}

// SourceFactory creates a Source from a SourceConfig
type SourceFactory func(config SourceConfig) (Source, error)

func NewSourceManager(db interface {
	SaveSource(source *SourceConfig) error
	GetSources() ([]SourceConfig, error)
	SaveTrack(track *Track) error
	GetTracks(sourceID string) ([]Track, error)
}) *SourceManager {
	return &SourceManager{
		db:              db,
		sources:         make(map[string]Source),
		sourceFactories: make(map[string]SourceFactory),
	}
}

// RegisterSourceType registers a new source type with its factory
func (m *SourceManager) RegisterSourceType(
	sourceType string,
	factory SourceFactory,
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sourceFactories[sourceType] = factory
}

// AddSource adds a new source with the given configuration and performs initial scan
func (m *SourceManager) AddSource(
	name, sourceType string,
	config map[string]string,
) error {
	m.mu.Lock()
	factory, ok := m.sourceFactories[sourceType]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("unknown source type: %s", sourceType)
	}

	sourceConfig := SourceConfig{
		ID:     uuid.NewString(),
		Type:   sourceType,
		Name:   name,
		Config: config,
	}

	source, err := factory(sourceConfig)
	if err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to create source: %w", err)
	}

	if err := m.db.SaveSource(&sourceConfig); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to save source: %w", err)
	}

	m.sources[sourceConfig.ID] = source
	m.mu.Unlock()

	// Perform initial scan
	return m.ScanSource(context.Background(), sourceConfig.ID)
}

// LoadSources loads all sources from the database
func (m *SourceManager) LoadSources() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sources, err := m.db.GetSources()
	if err != nil {
		return fmt.Errorf("failed to get sources: %w", err)
	}

	for _, config := range sources {
		factory, ok := m.sourceFactories[config.Type]
		if !ok {
			continue // Skip unknown source types
		}

		source, err := factory(config)
		if err != nil {
			continue // Skip sources that fail to initialize
		}

		m.sources[config.ID] = source
	}

	return nil
}

// ScanProgress represents the scanning progress
type ScanProgress struct {
	SourceID string
	Total    int
	Current  int
	Status   string
}

// GetScanProgress returns the current scanning progress
func (m *SourceManager) GetScanProgress() *ScanProgress {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scanProgress
}

// ScanSource scans a single source and updates the database
func (m *SourceManager) ScanSource(ctx context.Context, sourceID string) error {
	m.mu.RLock()
	source, ok := m.sources[sourceID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("source not found: %s", sourceID)
	}

	// Initialize progress
	m.mu.Lock()
	m.scanProgress = &ScanProgress{
		SourceID: sourceID,
		Status:   "Scanning...",
	}
	m.mu.Unlock()

	// Create buffered channels to avoid blocking
	tracks := make(chan Track, 100)
	errChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// Start track consumer
	go func() {
		defer close(doneChan)
		for track := range tracks {
			if err := m.db.SaveTrack(&track); err != nil {
				errChan <- fmt.Errorf("failed to save track: %w", err)
				return
			}
			m.mu.Lock()
			m.scanProgress.Current++
			m.mu.Unlock()
		}
	}()

	// Start source scanner
	go func() {
		if err := source.Scan(ctx, tracks, func(path string) {
			m.mu.Lock()
			m.scanProgress.Total++
			m.mu.Unlock()
		}); err != nil {
			errChan <- fmt.Errorf("scan failed: %w", err)
		}
	}()

	// Wait for completion or error
	select {
	case err := <-errChan:
		m.mu.Lock()
		m.scanProgress.Status = fmt.Sprintf("Error: %v", err)
		m.mu.Unlock()
		return err
	case <-doneChan:
		// Update last scanned time
		config := SourceConfig{
			ID:          sourceID,
			Type:        source.Type(),
			Name:        source.Name(),
			LastScanned: time.Now(),
		}
		if err := m.db.SaveSource(&config); err != nil {
			return fmt.Errorf("failed to update source: %w", err)
		}

		// Clear progress
		m.mu.Lock()
		m.scanProgress = nil
		m.mu.Unlock()
		return nil
	}
}

// GetSources returns all registered sources
func (m *SourceManager) GetSources() []SourceConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sources, err := m.db.GetSources()
	if err != nil {
		return nil
	}
	return sources
}

// GetTracks returns all tracks for a source
func (m *SourceManager) GetTracks(sourceID string) ([]Track, error) {
	return m.db.GetTracks(sourceID)
}
