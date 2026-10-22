// Package router fans game output out to any number of subscribers
// (TUI frontends, Python automations, ...).
package router

import (
	"sync"
	"time"

	pb "github.com/tollofsen/bazooka/gen/bazookapb"
)

const subscriberBuffer = 256

// Router is a publish/subscribe hub keyed by session ID.
type Router struct {
	mu   sync.RWMutex
	subs map[string]map[chan *pb.Output]struct{}
}

// New creates an empty Router.
func New() *Router {
	return &Router{subs: make(map[string]map[chan *pb.Output]struct{})}
}

// Subscribe returns a channel of output for a session and a cancel function.
// Slow subscribers drop messages rather than blocking the MUD reader.
func (r *Router) Subscribe(sessionID string) (<-chan *pb.Output, func()) {
	ch := make(chan *pb.Output, subscriberBuffer)
	r.mu.Lock()
	if r.subs[sessionID] == nil {
		r.subs[sessionID] = make(map[chan *pb.Output]struct{})
	}
	r.subs[sessionID][ch] = struct{}{}
	r.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			r.mu.Lock()
			delete(r.subs[sessionID], ch)
			if len(r.subs[sessionID]) == 0 {
				delete(r.subs, sessionID)
			}
			r.mu.Unlock()
		})
	}
}

// Publish delivers an output message to all subscribers of its session.
func (r *Router) Publish(o *pb.Output) {
	if o.TimestampUnixMs == 0 {
		o.TimestampUnixMs = time.Now().UnixMilli()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for ch := range r.subs[o.SessionId] {
		select {
		case ch <- o:
		default: // subscriber too slow; drop
		}
	}
}
