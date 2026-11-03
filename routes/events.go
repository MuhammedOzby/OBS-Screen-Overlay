package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"obs-screen-overlay/internal/live"
	"obs-screen-overlay/internal/store"
)

const (
	sseHeartbeatInterval = 15 * time.Second
	sseWriteDeadline     = 10 * time.Second
)

// serveEvents streams page state to an OBS page over SSE. Pre-stream errors
// map to 404 (unknown token) or 500; write and flush errors during the stream
// abort it so the client reconnects with a fresh snapshot.
func serveEvents(c *gin.Context, s *store.Store) {
	snap, ch, cancel, err := s.SubscribeLive(c.Request.Context(), c.Param("token"))
	if errors.Is(err, sql.ErrNoRows) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer cancel()

	// Best effort: keep slow clients from blocking the publisher. Servers
	// without write-deadline support ignore the error.
	ctrl := http.NewResponseController(c.Writer)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	send := func(s live.Snapshot) error {
		ctrl.SetWriteDeadline(time.Now().Add(sseWriteDeadline))
		data, err := json.Marshal(s)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "event: state\nid: %d\ndata: %s\n\n", s.Revision, data); err != nil {
			return err
		}
		return ctrl.Flush()
	}
	if err := send(snap); err != nil {
		return
	}

	tick := time.NewTicker(sseHeartbeatInterval)
	defer tick.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case s, ok := <-ch:
			if !ok { // Hub closed the channel (slow client dropped).
				return
			}
			if err := send(s); err != nil {
				return
			}
		case <-tick.C:
			ctrl.SetWriteDeadline(time.Now().Add(sseWriteDeadline))
			if _, err := c.Writer.WriteString(": heartbeat\n\n"); err != nil {
				return
			}
			if err := ctrl.Flush(); err != nil { return }
		}
	}
}
