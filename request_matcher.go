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
type Matcher func(r *Request, i int, rec []*Request) *Request

func defaultMatcher(r *Request, i int, rec []*Request) *Request {
	highScore := -math.MaxFloat64
	var bestMatch *Request
	faktor := float64(len(rec))
	for j, r := range rec {
		// TODO: this doesn't work so great rn
		if r.seen || r.FromOutside {
			continue
		}
		if j == i {
			log.Println("hello")
			bestMatch = r
			break
		}
		score := 0.0
		score -= math.Abs(float64(j - i))
		if r.To == r.To {
			score += 1 * faktor
		}
		if score > highScore {
			highScore = score
			bestMatch = r
		}
	}
	return bestMatch
}
