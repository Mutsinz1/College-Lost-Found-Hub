package api

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimit throttles a route to `requests` per `window`, keyed by client IP.
//
// This is a speed bump against casual abuse of the endpoints that accept
// unauthenticated writes -- creating posts, filing reports, subscribing to
// alerts -- not a defence against a determined or distributed attacker. Limits
// should stay generous: a campus network can put a whole building behind one
// address, and throttling real users is a worse outcome than the spam this
// prevents.
//
// The key comes from chi's RealIP middleware, so behind nginx this limits the
// actual client rather than lumping everyone onto the proxy's address. That
// also means RealIP must only run where X-Forwarded-For is set by a trusted
// proxy, which is the case in docker-compose.prod.yml.
//
// A limit of zero disables throttling for that route.
func RateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	if requests <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return httprate.Limit(
		requests,
		window,
		httprate.WithKeyFuncs(httprate.KeyByRealIP),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusTooManyRequests,
				"Too many requests from this network; please wait a little and try again", nil)
		}),
	)
}
