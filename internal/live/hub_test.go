package live

import (
	"strconv"
	"sync"
	"testing"
)

// TestSubscribe_ReceivesPublishedSnapshot verifies a subscriber receives the
// snapshot published on its page.
func TestSubscribe_ReceivesPublishedSnapshot(t *testing.T) {
	h := NewHub(4)
	ch, cancel := h.Subscribe("page-a")
	defer cancel()

	h.Publish("page-a", Snapshot{
		Revision:        3,
		TemplateVersion: 7,
		Fields:          map[string]string{"konuk_adi": "ÖZBAY"},
	})

	snap := <-ch
	if snap.Revision != 3 {
		t.Fatalf("revision = %d, want 3", snap.Revision)
	}
	if snap.TemplateVersion != 7 {
		t.Fatalf("template_version = %d, want 7", snap.TemplateVersion)
	}
	if got := snap.Fields["konuk_adi"]; got != "ÖZBAY" {
		t.Fatalf("fields[konuk_adi] = %q, want %q", got, "ÖZBAY")
	}
}

// TestIsolation_PagesAreIndependent verifies publishing one page does not
// deliver to subscribers of another page.
func TestIsolation_PagesAreIndependent(t *testing.T) {
	h := NewHub(4)
	a, cancelA := h.Subscribe("page-a")
	defer cancelA()
	b, cancelB := h.Subscribe("page-b")
	defer cancelB()

	h.Publish("page-a", Snapshot{Revision: 1})

	snap := <-a
	if snap.Revision != 1 {
		t.Fatalf("page-a revision = %d, want 1", snap.Revision)
	}

	// page-b must stay empty: page isolation means no cross-delivery.
	select {
	case got := <-b:
		t.Fatalf("page-b unexpectedly received snapshot %+v", got)
	default:
	}
}

// TestMultipleSubscribers_SamePage verifies all subscribers of a page get the
// snapshot exactly once.
func TestMultipleSubscribers_SamePage(t *testing.T) {
	h := NewHub(4)
	a, cancelA := h.Subscribe("page")
	defer cancelA()
	b, cancelB := h.Subscribe("page")
	defer cancelB()

	h.Publish("page", Snapshot{Revision: 5})

	if sn := <-a; sn.Revision != 5 {
		t.Fatalf("subscriber a revision = %d, want 5", sn.Revision)
	}
	if sn := <-b; sn.Revision != 5 {
		t.Fatalf("subscriber b revision = %d, want 5", sn.Revision)
	}
}

// TestOverflow_DropsSubscriber verifies that a slow subscriber whose channel
// overflows has its channel closed and is removed, while a healthy subscriber
// on the same page still receives the snapshot.
func TestOverflow_DropsSubscriber(t *testing.T) {
	h := NewHub(1) // capacity 1

	slow, cancelSlow := h.Subscribe("page")
	defer cancelSlow()
	fast, cancelFast := h.Subscribe("page")
	defer cancelFast()

	// First publish lands in both bounded channels (capacity 1 each).
	h.Publish("page", Snapshot{Revision: 1})

	// fast drains and stays ready; slow keeps its slot full (never read).
	if sn := <-fast; sn.Revision != 1 {
		t.Fatalf("fast revision = %d, want 1", sn.Revision)
	}

	// Second publish overflows slow (still full) but fits into ready fast.
	h.Publish("page", Snapshot{Revision: 2})

	// slow: first value buffered, then its channel is closed by the drop.
	if sn := <-slow; sn.Revision != 1 {
		t.Fatalf("slow revision = %d, want 1", sn.Revision)
	}
	select {
	case _, ok := <-slow:
		if ok {
			t.Fatalf("slow channel not closed after overflow")
		}
	default:
		t.Fatalf("slow channel closed but first snapshot not buffered")
	}

	// fast still receives the latest snapshot.
	if sn := <-fast; sn.Revision != 2 {
		t.Fatalf("fast revision = %d, want 2", sn.Revision)
	}
}

// TestCancel_Idempotent verifies calling cancel multiple times is safe and a
// publish afterwards does not panic on the previously closed channel.
func TestCancel_Idempotent(t *testing.T) {
	h := NewHub(4)
	_, cancel := h.Subscribe("page")

	cancel()
	cancel()
	cancel()

	// Publishing to an empty page must be a no-op, not a panic.
	h.Publish("page", Snapshot{Revision: 1})
}

// TestConcurrentRace exercises hub mutex safety and per-page isolation under
// concurrent subscribe, publish, and unsubscribe with -race.
func TestConcurrentRace(t *testing.T) {
	const pages = 8
	h := NewHub(2)

	var wg sync.WaitGroup
	for p := 0; p < pages; p++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "page" + strconv.Itoa(n)
			sub, cancel := h.Subscribe(id)
			defer cancel()
			for i := 0; i < 50; i++ {
				h.Publish(id, Snapshot{Revision: int64(i)})
				select {
				case <-sub:
				default:
				}
			}
		}(p)
	}
	wg.Wait()
}
