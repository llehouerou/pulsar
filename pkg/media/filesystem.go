package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/google/uuid"
)

type FilesystemSource struct {
	id        string
	name      string
	rootPaths []string
}

func NewFilesystemSource(id, name string, paths []string) *FilesystemSource {
	return &FilesystemSource{
		id:        id,
		name:      name,
		rootPaths: paths,
	}
}

func (fs *FilesystemSource) Type() string {
	return "filesystem"
}

func (fs *FilesystemSource) Name() string {
	return fs.name
}

func (fs *FilesystemSource) Scan(
	ctx context.Context,
	tracks chan<- Track,
	onFile func(path string),
) error {
	defer close(tracks)

	for _, rootPath := range fs.rootPaths {
		err := filepath.Walk(
			rootPath,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				if !info.IsDir() && onFile != nil {
					onFile(path)
				}

				if info.IsDir() || !isAudioFile(path) {
					return nil
				}

				track, err := fs.scanFile(path)
				if err != nil {
					return err
				}

				track.SourceID = fs.id
				track.SourceType = fs.Type()
				track.LastScanned = time.Now()

				tracks <- track
				return nil
			},
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *FilesystemSource) scanFile(path string) (Track, error) {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err == nil {
		defer tag.Close()
		return Track{
			ID:     uuid.NewString(),
			Path:   path,
			Title:  tag.Title(),
			Artist: tag.Artist(),
			Album:  tag.Album(),
		}, nil
	}

	// If we can't read tags, use filename as title
	return Track{
		ID:    uuid.NewString(),
		Path:  path,
		Title: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}, nil
}

func isAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp3"
}

// NewFilesystemSourceFactory creates a factory for filesystem sources
func NewFilesystemSourceFactory() SourceFactory {
	return func(config SourceConfig) (Source, error) {
		paths, ok := config.Config["paths"]
		if !ok {
			return nil, fmt.Errorf("missing paths in config")
		}

		// Split paths by semicolon
		pathList := strings.Split(paths, ";")
		return NewFilesystemSource(config.ID, config.Name, pathList), nil
	}
}
