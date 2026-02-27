package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != securityHeaderNoSniff {
		t.Fatalf("X-Content-Type-Options = %q", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("X-Frame-Options") != securityHeaderNoFrame {
		t.Fatalf("X-Frame-Options = %q", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("Strict-Transport-Security") != securityHeaderHSTS {
		t.Fatalf("Strict-Transport-Security = %q", rec.Header().Get("Strict-Transport-Security"))
	}
	if rec.Header().Get("Content-Security-Policy") != securityHeaderCSP {
		t.Fatalf("Content-Security-Policy = %q", rec.Header().Get("Content-Security-Policy"))
	}
}

func TestBodySizeLimit(t *testing.T) {
	handler := BodySizeLimit(8)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	reqSmall := httptest.NewRequest(http.MethodPost, "/v1/reports", strings.NewReader("12345678"))
	reqSmall.RemoteAddr = "203.0.113.10:12345"
	recSmall := httptest.NewRecorder()
	handler.ServeHTTP(recSmall, reqSmall)
	if recSmall.Code != http.StatusNoContent {
		t.Fatalf("small request status = %d, want %d", recSmall.Code, http.StatusNoContent)
	}

	reqLarge := httptest.NewRequest(http.MethodPost, "/v1/reports", strings.NewReader("123456789"))
	reqLarge.RemoteAddr = "203.0.113.10:12345"
	recLarge := httptest.NewRecorder()
	handler.ServeHTTP(recLarge, reqLarge)
	if recLarge.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large request status = %d, want %d", recLarge.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestRateLimitPerIP(t *testing.T) {
	handler := RateLimitPerIP(2, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
		req.RemoteAddr = "198.51.100.7:1000"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want %d", i+1, rec.Code, http.StatusNoContent)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.RemoteAddr = "198.51.100.7:1000"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limited status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitPerIPUsesForwardedFor(t *testing.T) {
	handler := RateLimitPerIP(1, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	first.RemoteAddr = "10.0.0.5:12345"
	first.Header.Set("X-Forwarded-For", "203.0.113.100, 10.0.0.5")
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)
	if firstRec.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d, want %d", firstRec.Code, http.StatusNoContent)
	}

	second := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	second.RemoteAddr = "10.0.0.6:12345"
	second.Header.Set("X-Forwarded-For", "203.0.113.100, 10.0.0.6")
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)
	if secondRec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", secondRec.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitPerIPWindowReset(t *testing.T) {
	current := time.Unix(0, 0)
	now := func() time.Time {
		return current
	}

	handler := rateLimitPerIPWithClock(1, time.Minute, now)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	first.RemoteAddr = "198.51.100.9:443"
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)
	if firstRec.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d, want %d", firstRec.Code, http.StatusNoContent)
	}

	second := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	second.RemoteAddr = "198.51.100.9:443"
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)
	if secondRec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", secondRec.Code, http.StatusTooManyRequests)
	}

	current = current.Add(time.Minute + time.Second)
	third := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	third.RemoteAddr = "198.51.100.9:443"
	thirdRec := httptest.NewRecorder()
	handler.ServeHTTP(thirdRec, third)
	if thirdRec.Code != http.StatusNoContent {
		t.Fatalf("third request status = %d, want %d", thirdRec.Code, http.StatusNoContent)
	}
}
