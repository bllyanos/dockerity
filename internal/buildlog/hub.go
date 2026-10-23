package buildlog

import "sync"

// Chunk is a piece of a build log starting at Offset bytes into the log.
// Offsets let a reader that loaded the persisted log skip what it has seen.
type Chunk struct {
	Offset int64
	Data   []byte
}

// subscriberBuffer bounds memory per live viewer. A viewer that falls this far
// behind misses chunks; the persisted log stays complete.
const subscriberBuffer = 256

// Hub fans live build log chunks out to subscribers.
type Hub struct {
	mu     sync.Mutex
	subs   map[int64]map[chan Chunk]struct{}
	closed map[int64]bool
}

// NewHub returns an empty hub.
func NewHub() *Hub {
	return &Hub{subs: map[int64]map[chan Chunk]struct{}{}, closed: map[int64]bool{}}
}

// Subscribe returns a channel of chunks for buildID and a cancel function.
// The channel is closed when the build's log ends or cancel is called. If the
// log already ended, the channel is closed immediately.
func (h *Hub) Subscribe(buildID int64) (<-chan Chunk, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan Chunk, subscriberBuffer)
	if h.closed[buildID] {
		close(ch)
		return ch, func() {}
	}
	if h.subs[buildID] == nil {
		h.subs[buildID] = map[chan Chunk]struct{}{}
	}
	h.subs[buildID][ch] = struct{}{}
	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, ok := h.subs[buildID][ch]; ok {
			delete(h.subs[buildID], ch)
			close(ch)
		}
	}
}

// Publish sends a chunk to every subscriber without blocking.
func (h *Hub) Publish(buildID int64, c Chunk) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[buildID] {
		select {
		case ch <- c:
		default: // slow viewer: drop rather than stall the build
		}
	}
}

// Close ends the build's log: subscribers' channels are closed and later
// subscribers get a closed channel.
func (h *Hub) Close(buildID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[buildID] {
		close(ch)
	}
	delete(h.subs, buildID)
	h.closed[buildID] = true
}

// Reopen forgets that a build's log ended, used when a build is retried.
func (h *Hub) Reopen(buildID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.closed, buildID)
}
