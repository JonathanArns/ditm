package main

import (
	"log"
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

type defaultMatcher struct {
	seen map[*Request]struct{}
}

func (m *defaultMatcher) Seen(r *Request) bool {
	_, ok := m.seen[r]
	return ok
}

func (m *defaultMatcher) MarkSeen(r *Request) {
	m.seen[r] = struct{}{}
}

func (m *defaultMatcher) Match(r *Request, i int, rec []*Request) *Request {
	highScore := -math.MaxFloat64
	var bestMatch *Request
	faktor := float64(len(rec))
	for j, req := range rec {
		// TODO: this doesn't work so great rn
		if m.Seen(req) || req.FromOutside {
			continue
		}
		if j == i {
			log.Println("hello")
			bestMatch = req
			break
		}
		score := 0.0
		score -= math.Abs(float64(j - i))
		if req.To == r.To {
			score += 1 * faktor
		}
		if score > highScore {
			highScore = score
			bestMatch = req
		}
	}
	return bestMatch
}

type smartMatcher struct {
	seen map[*Request]struct{}
}

func (m *smartMatcher) Seen(r *Request) bool {
	_, ok := m.seen[r]
	return ok
}

func (m *smartMatcher) MarkSeen(r *Request) {
	m.seen[r] = struct{}{}
}

func (m *smartMatcher) Match(r *Request, i int, rec []*Request) *Request {
	highScore := -math.MaxFloat64
	var bestMatch *Request
	faktor := float64(len(rec))
	for j, req := range rec {
		if m.Seen(req) || req.FromOutside {
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

type simpleMatcher struct {
	seen map[*Request]struct{}
}

func (m *simpleMatcher) Seen(r *Request) bool {
	_, ok := m.seen[r]
	return ok
}

func (m *simpleMatcher) MarkSeen(r *Request) {
	m.seen[r] = struct{}{}
}

func (m *simpleMatcher) Match(r *Request, i int, rec []*Request) *Request {
	highScore := -math.MaxFloat64
	var bestMatch *Request
	faktor := float64(len(rec))
	for j, req := range rec {
		if m.Seen(req) || req.FromOutside {
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
