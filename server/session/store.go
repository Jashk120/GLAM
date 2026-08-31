package session

import (
	"sync"

	"glam/server/llm"
)

type Store struct {
	mu       sync.Mutex
	sessions map[string][]llm.Message
}

func NewStore() *Store {
	return &Store{sessions: make(map[string][]llm.Message)}
}

func (s *Store) Get(id string) []llm.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := s.sessions[id]
	out := make([]llm.Message, len(msgs))
	copy(out, msgs)
	return out
}

func (s *Store) Set(id string, msgs []llm.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]llm.Message, len(msgs))
	copy(cp, msgs)
	s.sessions[id] = cp
}

func (s *Store) Append(id string, msgs ...llm.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = append(s.sessions[id], msgs...)
}

func (s *Store) Exists(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[id]
	return ok
}
