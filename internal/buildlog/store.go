// Package buildlog persists build logs as one append-only file per build.
package buildlog

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// DefaultMaxBytes caps a single build log.
const DefaultMaxBytes = 10 << 20

const truncatedNotice = "\n[dockerity] log truncated: size limit reached\n"

// Store keeps build logs under Dir.
type Store struct {
	Dir      string
	MaxBytes int64
}

func (s Store) path(buildID int64) string {
	return filepath.Join(s.Dir, strconv.FormatInt(buildID, 10)+".log")
}

// Open returns a writer appending to the build's log. Writes beyond the size
// cap are dropped after a single truncation notice.
func (s Store) Open(buildID int64) (io.WriteCloser, error) {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(s.path(buildID), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	limit := s.MaxBytes
	if limit <= 0 {
		limit = DefaultMaxBytes
	}
	return &cappedWriter{f: f, written: info.Size(), limit: limit, truncated: info.Size() >= limit}, nil
}

// Read returns the whole log, or an empty log if nothing was written yet.
func (s Store) Read(buildID int64) ([]byte, error) {
	data, err := os.ReadFile(s.path(buildID))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

// Remove deletes a build's log.
func (s Store) Remove(buildID int64) error {
	err := os.Remove(s.path(buildID))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

type cappedWriter struct {
	mu        sync.Mutex
	f         *os.File
	written   int64
	limit     int64
	truncated bool
}

// Write never fails because of the cap: callers keep streaming, the excess is dropped.
func (w *cappedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.truncated {
		return len(p), nil
	}
	room := w.limit - w.written
	if int64(len(p)) > room {
		if room > 0 {
			if _, err := w.f.Write(p[:room]); err != nil {
				return 0, err
			}
		}
		if _, err := io.WriteString(w.f, truncatedNotice); err != nil {
			return 0, err
		}
		w.truncated = true
		w.written = w.limit
		return len(p), nil
	}
	n, err := w.f.Write(p)
	w.written += int64(n)
	return n, err
}

func (w *cappedWriter) Close() error { return w.f.Close() }
