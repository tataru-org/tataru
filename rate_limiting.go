package main

import (
	"math"
	"math/rand"
	"time"
)

func RandomRange(min float64, max float64) float64 {
	rand.Seed(time.Now().UnixNano())
	return (max-min)*rand.Float64() + min
}

func CalcWaitDuration(rateLimit float64) float64 {
	return 1 / rateLimit * (1 + RandomRange(0, 1))
}

func CalcThrottledWaitDuration(rateLimit, maxWaitDuration float64) float64 {
	jitter := RandomRange(0, rateLimit)
	return math.Min(maxWaitDuration, math.Pow(rateLimit, 3)+jitter)
}
