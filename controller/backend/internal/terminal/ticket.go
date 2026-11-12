package terminal

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

var errTooManyTickets = errors.New("too many unused tickets")

// ticket is what a one-time terminal ticket stands for. The browser gets it
// over REST (with its Bearer JWT) and hands it back as a WebSocket
// subprotocol, because a browser WebSocket cannot send an Authorization header.
type ticket struct {
	user     string
	runner   string
	remoteIP string

	cols uint16
	rows uint16

	// tokenExpires is the expiry of the JWT the ticket was issued for;
	// the session ends then at the latest.
	tokenExpires time.Time

	expires time.Time
}

type ticketStore struct {
	mu        sync.Mutex
	ttl       time.Duration
	maxByUser int
	max       int
	tickets   map[string]ticket
	now       func() time.Time
}

func newTicketStore(ttl time.Duration, maxByUser, max int, now func() time.Time) *ticketStore {
	return &ticketStore{
		ttl:       ttl,
		maxByUser: maxByUser,
		max:       max,
		tickets:   make(map[string]ticket),
		now:       now,
	}
}

// issue stores t and returns its id: 32 random bytes, base64url without
// padding, which is a valid subprotocol token.
func (s *ticketStore) issue(t ticket) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(b)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	byUser := 0
	for k, v := range s.tickets {
		if !now.Before(v.expires) {
			delete(s.tickets, k)
			continue
		}
		if v.user == t.user {
			byUser++
		}
	}
	if byUser >= s.maxByUser || len(s.tickets) >= s.max {
		return "", errTooManyTickets
	}

	t.expires = now.Add(s.ttl)
	s.tickets[id] = t

	return id, nil
}

// consume returns the ticket and removes it, so a ticket works only once.
func (s *ticketStore) consume(id string) (ticket, bool) {
	if id == "" {
		return ticket{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[id]
	if !ok {
		return ticket{}, false
	}
	delete(s.tickets, id)

	if !s.now().Before(t.expires) {
		return ticket{}, false
	}

	return t, true
}
