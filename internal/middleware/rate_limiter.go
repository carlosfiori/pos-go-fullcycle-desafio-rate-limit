package middleware

import (
	"context"
	"net"
	"net/http"
)

type Limiter interface {
	Allow(ctx context.Context, ip string, token string) (bool, error)
}

func RateLimiter(rl Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {

				ip = r.RemoteAddr
			}

			token := r.Header.Get("API_KEY")

			allowed, err := rl.Allow(r.Context(), ip, token)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "you have reached the maximum number of requests or actions allowed within a certain time frame", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
