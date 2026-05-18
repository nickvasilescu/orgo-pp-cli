// Copyright 2026 nickvasilescu. Licensed under Apache-2.0. See LICENSE.

package cliutil

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestAuthErrorHelpers(t *testing.T) {
	if !LooksLikeAuthError("HTTP 400: missing api_key") {
		t.Fatal("expected missing api_key to look like an auth error")
	}
	if LooksLikeAuthError("HTTP 400: malformed page number") {
		t.Fatal("unexpected auth classification for non-auth message")
	}

	got := SanitizeErrorBody("token sk-abcdefghi Bearer abc.def key=secretvalue")
	if got != "token [REDACTED] [REDACTED] [REDACTED]" {
		t.Fatalf("SanitizeErrorBody redaction = %q", got)
	}
}

func TestAdaptiveLimiter_NewNilOnNonPositive(t *testing.T) {
	if NewAdaptiveLimiter(0) != nil {
		t.Fatal("NewAdaptiveLimiter(0) should return nil")
	}
	if NewAdaptiveLimiter(-1) != nil {
		t.Fatal("NewAdaptiveLimiter(-1) should return nil")
	}
}

func TestAdaptiveLimiter_NilSafeMethods(t *testing.T) {
	var l *AdaptiveLimiter
	l.Wait()
	l.OnSuccess()
	l.OnRateLimit()
	if got := l.Rate(); got != 0 {
		t.Errorf("nil limiter Rate() = %v, want 0", got)
	}
}

func TestAdaptiveLimiter_RampsUpAfterSuccesses(t *testing.T) {
	l := NewAdaptiveLimiter(2.0)
	startRate := l.Rate()
	for i := 0; i < l.rampAfter; i++ {
		l.OnSuccess()
	}
	if got := l.Rate(); got <= startRate {
		t.Errorf("Rate() after rampAfter successes = %v, want > %v", got, startRate)
	}
}

func TestAdaptiveLimiter_HalvesOnRateLimit(t *testing.T) {
	l := NewAdaptiveLimiter(8.0)
	startRate := l.Rate()
	l.OnRateLimit()
	got := l.Rate()
	if got != startRate/2 {
		t.Errorf("Rate() after OnRateLimit = %v, want %v", got, startRate/2)
	}
}

func TestAdaptiveLimiter_FloorsAtHalfRPS(t *testing.T) {
	l := NewAdaptiveLimiter(2.0)
	for i := 0; i < 10; i++ {
		l.OnRateLimit()
	}
	if got := l.Rate(); got < 0.5 {
		t.Errorf("Rate() after many OnRateLimit = %v, want >= 0.5", got)
	}
}

func TestAdaptiveLimiter_WaitEnforcesPacing(t *testing.T) {
	l := NewAdaptiveLimiter(10.0)
	l.Wait()
	start := time.Now()
	l.Wait()
	elapsed := time.Since(start)
	if elapsed < 80*time.Millisecond {
		t.Errorf("second Wait() took %v, want >= 80ms", elapsed)
	}
}

func TestRetryAfter_Seconds(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "10")
	if got := RetryAfter(resp); got != 10*time.Second {
		t.Errorf("RetryAfter(10) = %v, want 10s", got)
	}
}

func TestRetryAfter_HTTPDate(t *testing.T) {
	future := time.Now().Add(7 * time.Second).UTC().Format(http.TimeFormat)
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", future)
	got := RetryAfter(resp)
	if got < 5*time.Second || got > 8*time.Second {
		t.Errorf("RetryAfter(http-date 7s ahead) = %v, want ~7s", got)
	}
}

func TestRetryAfter_EpochSeconds(t *testing.T) {
	future := time.Now().Add(7 * time.Second)
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", fmt.Sprint(future.Unix()))
	got := RetryAfter(resp)
	if got < 5*time.Second || got > 8*time.Second {
		t.Errorf("RetryAfter(epoch seconds 7s ahead) = %v, want ~7s", got)
	}
}

func TestRetryAfter_EpochMilliseconds(t *testing.T) {
	future := time.Now().Add(7 * time.Second)
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", fmt.Sprint(future.UnixMilli()))
	got := RetryAfter(resp)
	if got < 5*time.Second || got > 8*time.Second {
		t.Errorf("RetryAfter(epoch milliseconds 7s ahead) = %v, want ~7s", got)
	}
}

func TestRetryAfter_Cap(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "600")
	if got := RetryAfter(resp); got != MaxRetryWait {
		t.Errorf("RetryAfter(600) = %v, want capped at %v", got, MaxRetryWait)
	}
}

func TestRetryAfter_Missing(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	if got := RetryAfter(resp); got != 5*time.Second {
		t.Errorf("RetryAfter(missing) = %v, want 5s default", got)
	}
}

func TestRetryAfter_MalformedFallsBackToDefault(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "not-a-number")
	if got := RetryAfter(resp); got != 5*time.Second {
		t.Errorf("RetryAfter(garbage) = %v, want 5s default", got)
	}
}

func TestRetryAfter_NilResp(t *testing.T) {
	if got := RetryAfter(nil); got != 5*time.Second {
		t.Errorf("RetryAfter(nil) = %v, want 5s default", got)
	}
}
