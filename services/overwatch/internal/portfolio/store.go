package portfolio

import "sync"

const maxHistory = 200

type Store struct {
	mu          sync.RWMutex
	insights    []Insight
	podInsights map[string]PodInsight // "namespace/app" → PodInsight
}

func NewStore() *Store {
	return &Store{podInsights: make(map[string]PodInsight)}
}

func (s *Store) AddInsight(ins Insight) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.insights) >= maxHistory {
		s.insights = s.insights[1:]
	}
	s.insights = append(s.insights, ins)
}

func (s *Store) GetLatest() (Insight, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.insights) == 0 {
		return Insight{}, false
	}
	return s.insights[len(s.insights)-1], true
}

func (s *Store) GetHistory(limit int) []Insight {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.insights)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]Insight, limit)
	copy(out, s.insights[n-limit:])
	return out
}

func (s *Store) SetPodInsight(namespace, app string, p PodInsight) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.podInsights[namespace+"/"+app] = p
}

func (s *Store) GetPodInsight(namespace, app string) (PodInsight, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.podInsights[namespace+"/"+app]
	return p, ok
}

func (s *Store) GetAllPodInsights() []PodInsight {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PodInsight, 0, len(s.podInsights))
	for _, p := range s.podInsights {
		out = append(out, p)
	}
	return out
}
