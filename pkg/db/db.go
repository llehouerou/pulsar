package db

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/llehouerou/pulsar/pkg/media"
)

type DB struct {
	db *sql.DB
}

func New(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS sources (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			config TEXT NOT NULL,
			last_scanned DATETIME
		);

		CREATE TABLE IF NOT EXISTS tracks (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			path TEXT NOT NULL,
			title TEXT NOT NULL,
			artist TEXT,
			album TEXT,
			duration INTEGER,
			last_scanned DATETIME,
			FOREIGN KEY(source_id) REFERENCES sources(id)
		);

		CREATE INDEX IF NOT EXISTS idx_tracks_source ON tracks(source_id);
		CREATE INDEX IF NOT EXISTS idx_tracks_artist ON tracks(artist);
		CREATE INDEX IF NOT EXISTS idx_tracks_album ON tracks(album);
	`)
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (d *DB) SaveSource(source *media.SourceConfig) error {
	config, err := json.Marshal(source.Config)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		INSERT OR REPLACE INTO sources (id, type, name, config, last_scanned)
		VALUES (?, ?, ?, ?, ?)
	`, source.ID, source.Type, source.Name, string(config), source.LastScanned)
	return err
}

func (d *DB) GetSources() ([]media.SourceConfig, error) {
	rows, err := d.db.Query(`
		SELECT id, type, name, config, last_scanned
		FROM sources
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []media.SourceConfig
	for rows.Next() {
		var source media.SourceConfig
		var configStr string
		err := rows.Scan(
			&source.ID,
			&source.Type,
			&source.Name,
			&configStr,
			&source.LastScanned,
		)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal([]byte(configStr), &source.Config)
		if err != nil {
			return nil, err
		}

		sources = append(sources, source)
	}
	return sources, nil
}

func (d *DB) SaveTrack(track *media.Track) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO tracks (
			id, source_id, source_type, path, title, artist, album,
			duration, last_scanned
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, track.ID, track.SourceID, track.SourceType, track.Path,
		track.Title, track.Artist, track.Album,
		track.Duration.Milliseconds(), track.LastScanned)
	return err
}

func (d *DB) GetTracks(sourceID string) ([]media.Track, error) {
	rows, err := d.db.Query(`
		SELECT id, source_id, source_type, path, title, artist, album,
			duration, last_scanned
		FROM tracks
		WHERE source_id = ?
		ORDER BY artist, album, title
	`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []media.Track
	for rows.Next() {
		var track media.Track
		var durationMs int64
		err := rows.Scan(
			&track.ID, &track.SourceID, &track.SourceType,
			&track.Path, &track.Title, &track.Artist, &track.Album,
			&durationMs, &track.LastScanned,
		)
		if err != nil {
			return nil, err
		}
		track.Duration = time.Duration(durationMs) * time.Millisecond
		tracks = append(tracks, track)
	}
	return tracks, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) SaveSetting(key, value string) error {
	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO settings (key, value)
		VALUES (?, ?)
	`, key, value)
	return err
}

func (d *DB) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow(`
		SELECT value FROM settings
		WHERE key = ?
	`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}
