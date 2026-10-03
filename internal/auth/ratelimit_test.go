package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPRateLimiter_BlocksAfterBurst(t *testing.T) {
	limiter := NewIPRateLimiter(0, 2, false)

	called := 0
	handler := limiter.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	}))

	newRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "203.0.113.1:1234"
		return req
	}

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newRequest())
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newRequest())
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, 2, called)
}

func TestIPRateLimiter_TrustProxyUsesForwardedFor(t *testing.T) {
	limiter := NewIPRateLimiter(0, 1, true)

	handler := limiter.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := func(forwardedFor string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = "10.0.0.1:1234" // Caddy's address, same for every client
		r.Header.Set("X-Forwarded-For", forwardedFor)
		return r
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req("198.51.100.1"))
	assert.Equal(t, http.StatusOK, rec.Code)

	// Same forwarded IP again: burst of 1 is exhausted.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req("198.51.100.1"))
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// A different forwarded IP gets its own bucket.
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req("198.51.100.2"))
	assert.Equal(t, http.StatusOK, rec.Code)
}
