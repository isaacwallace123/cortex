// Package application contains the Nerva event bus logic.
package application

import (
	"sync"

	"github.com/isaacwallace123/cortex/services/nerva/internal/domain"
)

// Bus is the in-memory fan-out broadcaster.
// Each Subscribe call registers a channel; Publish writes to all of them.
// Slow consumers get a dropped event (non-blocking send) rather than back-pressure.
type Bus struct {
	mu   sync.RWMutex
	subs map[string]chan *domain.Event
}

func NewBus() *Bus {
	return &Bus{subs: make(map[string]chan *domain.Event)}
}

// Publish fans out event to every active subscriber whose filter matches.
func (b *Bus) Publish(event *domain.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- event:
		default: // subscriber is slow — drop rather than block
		}
	}
}

// Subscribe registers a new subscriber and returns its channel and a cleanup func.
func (b *Bus) Subscribe(id string) (<-chan *domain.Event, func()) {
	ch := make(chan *domain.Event, 64)
	b.mu.Lock()
	b.subs[id] = ch
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		delete(b.subs, id)
		b.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}
