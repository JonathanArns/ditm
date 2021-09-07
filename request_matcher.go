package main

import (
	"log"
	"math"
)

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
