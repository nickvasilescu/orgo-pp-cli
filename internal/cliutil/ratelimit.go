// Copyright 2026 nickvasilescu. Licensed under Apache-2.0. See LICENSE.

package cliutil

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AdaptiveLimiter paces outbound requests with adaptive ceiling discovery.
// Starts at a floor rate, ramps up after consecutive successes, halves on 429
// and records a ceiling. Per-session only — not persisted. Methods are safe
// to call on a nil receiver.
type AdaptiveLimiter struct {
	mu          sync.Mutex
	rate        float64
	floor       float64
	ceiling     float64
	successes   int
	rampAfter   int
	lastRequest time.Time // zero-value: first Wait() returns immediately
}

// NewAdaptiveLimiter returns a limiter starting at ratePerSec, or nil when
// rate-limiting should be disabled. Methods on the nil limiter no-op.
func NewAdaptiveLimiter(ratePerSec float64) *AdaptiveLimiter {
	if ratePerSec <= 0 {
		return nil
	}
	return &AdaptiveLimiter{
		rate:      ratePerSec,
		floor:     ratePerSec,
		rampAfter: 10,
	}
}

func (l *AdaptiveLimiter) Wait() {
	if l == nil {
		return
	}
	l.mu.Lock()
	delay := time.Duration(float64(time.Second) / l.rate)
	elapsed := time.Since(l.lastRequest)
	l.mu.Unlock()
	if elapsed < delay {
		time.Sleep(delay - elapsed)
	}
	l.mu.Lock()
	l.lastRequest = time.Now()
	l.mu.Unlock()
}

func (l *AdaptiveLimiter) OnSuccess() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.successes++
	if l.successes >= l.rampAfter {
		newRate := l.rate * 1.25
		if l.ceiling > 0 && newRate > l.ceiling*0.9 {
			newRate = l.ceiling * 0.9
		}
		l.rate = newRate
		l.successes = 0
	}
}

func (l *AdaptiveLimiter) OnRateLimit() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ceiling = l.rate
	l.rate = l.rate / 2
	if l.rate < 0.5 {
		l.rate = 0.5
	}
	l.successes = 0
}

func (l *AdaptiveLimiter) Rate() float64 {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rate
}

// MaxRetryWait caps the wait derived from a Retry-After header so a buggy
// or hostile upstream cannot pin a CLI for hours.
const MaxRetryWait = 60 * time.Second

const (
	defaultRetryWait               = 5 * time.Second
	unixEpochSecondsThreshold      = 1_000_000_000
	unixEpochMillisecondsThreshold = 1_000_000_000_000
)

// RetryAfter parses an HTTP Retry-After header (RFC 7231: delta-seconds or
// HTTP-date), plus common Unix epoch seconds/milliseconds variants emitted by
// some APIs. Waits are capped at MaxRetryWait. Returns 5s when missing or
// unparseable.
func RetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return defaultRetryWait
	}
	header := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if header == "" {
		return defaultRetryWait
	}
	if value, err := strconv.ParseInt(header, 10, 64); err == nil {
		return retryAfterFromNumber(value)
	}
	if t, err := http.ParseTime(header); err == nil {
		wait := time.Until(t)
		if wait > MaxRetryWait {
			return MaxRetryWait
		}
		if wait > 0 {
			return wait
		}
	}
	return defaultRetryWait
}

func retryAfterFromNumber(value int64) time.Duration {
	if value <= 0 {
		return defaultRetryWait
	}
	if value > int64(MaxRetryWait/time.Second) {
		if wait := retryAfterEpochWait(value); wait > 0 {
			if wait > MaxRetryWait {
				return MaxRetryWait
			}
			return wait
		}
		return MaxRetryWait
	}
	return time.Duration(value) * time.Second
}

func retryAfterEpochWait(value int64) time.Duration {
	switch {
	case value >= unixEpochMillisecondsThreshold:
		return time.Until(time.UnixMilli(value))
	case value >= unixEpochSecondsThreshold:
		return time.Until(time.Unix(value, 0))
	default:
		return 0
	}
}
