package live

import "sync"

// Snapshot is the immutable state delivered to subscribers. It mirrors the
// JSON payload sent over SSE for a page.
type Snapshot struct {
	Revision        int64             `json:"revision"`
	TemplateVersion int64             `json:"template_version"`
	Fields          map[string]string `json:"fields"`
}

// subscriber represents a single receive-side channel for one page.
type subscriber struct {
	ch chan Snapshot
}

// Hub is a page-keyed pub/sub hub. It holds a mutex (h.mu), a per-subscriber
// channel buffer (h.buf), and a map (h.subs) that for each pageID maintains a
// set of registered subscriber pointers (*subscriber).
type Hub struct {
	mu   sync.Mutex
	buf  int
	subs map[string]map[*subscriber]struct{}
}

// NewHub returns a Hub whose per-subscriber channel has the given capacity.
// A capacity below 1 is raised to 1. The per-page subscriber map is
// initialized so the hub can be used immediately.
func NewHub(buf int) *Hub {
	if buf < 1 {
		buf = 1
	}
	return &Hub{
		buf:  buf,
		subs: make(map[string]map[*subscriber]struct{}),
	}
}

// Subscribe registers a new buffered subscriber for pageID and returns a
// receive-only channel plus an idempotent cancellation function. The channel
// has capacity h.buf. The returned cancel function removes the subscriber and
// closes its channel exactly once; calling it more than once (or calling it
// after the page was closed or the subscriber was dropped) is a no-op.
func (h *Hub) Subscribe(pageID string) (<-chan Snapshot, func()) {
	h.mu.Lock()
	page, ok := h.subs[pageID]
	if !ok {
		page = make(map[*subscriber]struct{})
		h.subs[pageID] = page
	}
	s := &subscriber{ch: make(chan Snapshot, h.buf)}
	page[s] = struct{}{}
	h.mu.Unlock()
	return s.ch, func() {
		h.unsubscribe(pageID, s)
	}
}

// unsubscribe locks the hub and, only if the subscriber is still registered
// for pageID, closes its channel and removes it from the page. If the page
// becomes empty the page entry is dropped. Calling it on an already-removed
// subscriber is a no-op, which makes the cancel function idempotent and
// guarantees the channel is never closed twice.
func (h *Hub) unsubscribe(pageID string, s *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeSubscriber(pageID, s)
}

// removeSubscriber removes s from the page and closes its channel, but only
// while the subscriber is still registered. It requires the caller to already
// hold h.mu. It is a no-op if s is not currently registered, which prevents
// double-closing the channel.
func (h *Hub) removeSubscriber(pageID string, s *subscriber) {
	page, ok := h.subs[pageID]
	if !ok {
		return
	}
	if _, registered := page[s]; !registered {
		return
	}
	delete(page, s)
	close(s.ch)
	if len(page) == 0 {
		delete(h.subs, pageID)
	}
}

// Publish sends snap to every subscriber of pageID using non-blocking sends
// while holding the hub lock. A subscriber whose channel is full is dropped:
// its channel is closed and it is removed from the page so a slow client stops
// blocking the publisher and can reconnect with a fresh snapshot. Empty page
// maps are removed from the hub.
func (h *Hub) Publish(pageID string, snap Snapshot) {
	h.mu.Lock()
	page, ok := h.subs[pageID]
	if !ok {
		h.mu.Unlock()
		return
	}
	for s := range page {
		select {
		case s.ch <- snap:
		default:
			// Slow subscriber: its buffer is full, drop and close it.
			h.removeSubscriber(pageID, s)
		}
	}
	h.mu.Unlock()
}

// ClosePage locks the hub, closes the channel of every subscriber registered for
// pageID, then deletes the page entry. It is a no-op for unknown or already
// empty pages.
func (h *Hub) ClosePage(pageID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	page, ok := h.subs[pageID]
	if !ok {
		return
	}
	for s := range page {
		close(s.ch)
	}
	delete(h.subs, pageID)
}
