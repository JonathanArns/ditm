package main

import (
	"math"
)

// Ideas for matcher heuristics:

// Position in stream
// timing
// URI
// length of body
// length of URI
// names of the query params
// length of individual query params

// try to also make a metric with only data that would be available on tcp

// A Matcher returns a pointer to the Request from rec,
// that matches r the best.
// If no match is found, nil is returned.
// i is the index of r in its recording.
type Matcher interface {
	Match(r *Request, i int, rec []*Request) *Request
	Seen(r *Request) bool
	MarkSeen(r *Request)
}

type countingMatcher struct {
	seen map[*Request]struct{}
}

func (m *countingMatcher) Seen(r *Request) bool {
	_, ok := m.seen[r]
	return ok
}

func (m *countingMatcher) MarkSeen(r *Request) {
	m.seen[r] = struct{}{}
}

func (m *countingMatcher) Match(r *Request, i int, rec []*Request) *Request {
	if i < len(rec) {
		m.MarkSeen(rec[i])
		return rec[i]
	}
	return nil
}

type heuristicMatcher struct {
	seen map[*Request]struct{}
}

func (m *heuristicMatcher) Seen(r *Request) bool {
	_, ok := m.seen[r]
	return ok
}

func (m *heuristicMatcher) MarkSeen(r *Request) {
	m.seen[r] = struct{}{}
}

func (m *heuristicMatcher) Match(r *Request, i int, rec []*Request) *Request {
	highScore := -math.MaxFloat64
	var bestMatch *Request
	faktor := float64(len(rec))
	for j, req := range rec {
		if m.Seen(req) || req.FromOutside || r.Method != req.Method {
			continue
		}
		score := 0.0
		score -= math.Abs(float64(j - i))
		if req.To == r.To {
			score += 1 * faktor
		}
		score -= math.Abs(float64(len(req.To)-len(r.To))) * faktor / 10
		score -= math.Abs(float64(len(req.Body)-len(r.Body))) * faktor / 10
		if score > highScore {
			highScore = score
			bestMatch = req
		}
	}
	return bestMatch
}

type exactMatcher struct {
	seen map[*Request]struct{}
}

func (m *exactMatcher) Seen(r *Request) bool {
	_, ok := m.seen[r]
	return ok
}

func (m *exactMatcher) MarkSeen(r *Request) {
	m.seen[r] = struct{}{}
}

func (m *exactMatcher) Match(r *Request, i int, rec []*Request) *Request {
	highScore := -math.MaxFloat64
	var bestMatch *Request
	for j, req := range rec {
		if m.Seen(req) || req.FromOutside || r.Method != req.Method {
			continue
		}
		score := 0.0
		if i == j {
			score += 1
		}
		if req.To == r.To {
			score += 1
		}
		if len(req.To) == len(r.To) {
			score += 1
		}
		if string(req.Body) == string(r.Body) {
			score += 1
		}
		if len(req.Body) == len(r.Body) {
			score += 1
		}
		if score > highScore {
			highScore = score
			bestMatch = req
		}
	}
	return bestMatch
}

type mixMatcher struct {
	seen map[*Request]struct{}
}

func (m *mixMatcher) Seen(r *Request) bool {
	_, ok := m.seen[r]
	return ok
}

func (m *mixMatcher) MarkSeen(r *Request) {
	m.seen[r] = struct{}{}
}

func (m *mixMatcher) Match(r *Request, i int, rec []*Request) *Request {
	highScore := -math.MaxFloat64
	var bestMatch *Request
	faktor := float64(len(rec)) / 10
	for j, req := range rec {
		if m.Seen(req) || req.FromOutside || r.Method != req.Method {
			continue
		}
		score := 0.0
		score -= math.Abs(float64(j - i))
		if req.To == r.To {
			score += 1 * faktor
		}
		if len(req.To) == len(r.To) {
			score += 1 * faktor
		}
		if string(req.Body) == string(r.Body) {
			score += 1 * faktor
		}
		if len(req.Body) == len(r.Body) {
			score += 1 * faktor
		}
		if score > highScore {
			highScore = score
			bestMatch = req
		}
	}
	return bestMatch
}
