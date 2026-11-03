package store

import (
	"context"
	"fmt"

	"obs-screen-overlay/internal/live"
)

// SubscribeLive loads the page identified by the public token together with
// its current snapshot and subscribes to the live hub for that page, all
// while holding liveMu. Holding liveMu across the read-and-subscribe window
// guarantees a consistent snapshot: no committed change is lost between
// reading the state and starting to receive it.
//
// It returns the current snapshot, a receive-only channel of subsequent
// snapshots, a cancel function (idempotent; safe to call more than once),
// and an error if the page cannot be found or the hub is unavailable.
func (s *Store) SubscribeLive(ctx context.Context, token string) (live.Snapshot, <-chan live.Snapshot, func(), error) {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()

	if s.hub == nil {
		return live.Snapshot{}, nil, nil, fmt.Errorf("canlı yayın merkezi başlatılmamış")
	}

	page, err := s.GetPageByPublicToken(ctx, token)
	if err != nil {
		return live.Snapshot{}, nil, nil, fmt.Errorf("page bulunamadı: %w", err)
	}

	// Snapshot: only SSE-enabled fields, per the contract in AGENT.md §5.
	fields := make(map[string]string, len(page.Fields))
	for _, f := range page.Fields {
		if !f.SSEEnabled {
			continue
		}
		fields[f.Name] = f.Value
	}

	snap := live.Snapshot{
		Revision:        page.Revision,
		TemplateVersion: page.TemplateVersion,
		Fields:          fields,
	}

	ch, cancel := s.hub.Subscribe(page.ID)
	return snap, ch, cancel, nil
}
