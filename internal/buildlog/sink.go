package buildlog

import (
	"io"
	"sync"
)

// Sink writes a build's log to its file and publishes it live.
type Sink struct {
	mu      sync.Mutex
	buildID int64
	file    io.WriteCloser
	hub     *Hub
	offset  int64
}

// OpenSink opens the build's log for writing and live publishing. The offset
// starts at the current file size, so chunks line up with the persisted log.
func (s Store) OpenSink(buildID int64, hub *Hub) (*Sink, error) {
	existing, err := s.Read(buildID)
	if err != nil {
		return nil, err
	}
	f, err := s.Open(buildID)
	if err != nil {
		return nil, err
	}
	return &Sink{buildID: buildID, file: f, hub: hub, offset: int64(len(existing))}, nil
}

// Write appends p to the file and publishes it.
func (s *Sink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.file.Write(p)
	if n > 0 {
		s.hub.Publish(s.buildID, Chunk{Offset: s.offset, Data: append([]byte(nil), p[:n]...)})
		s.offset += int64(n)
	}
	return n, err
}

// Close closes the file and ends the live stream.
func (s *Sink) Close() error {
	err := s.file.Close()
	s.hub.Close(s.buildID)
	return err
}

// Since returns the part of chunk c that lies at or after offset seen, or nil.
func Since(c Chunk, seen int64) []byte {
	end := c.Offset + int64(len(c.Data))
	switch {
	case end <= seen:
		return nil
	case c.Offset >= seen:
		return c.Data
	default:
		return c.Data[seen-c.Offset:]
	}
}
