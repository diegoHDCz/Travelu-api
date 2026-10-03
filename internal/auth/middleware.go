package auth

import (
	"net/http"
	"strings"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

const bearerPrefix = "Bearer "

// RequireAuth validates the Authorization: Bearer <token> header and stores
// the authenticated user ID in the request context (see UserIDFrom).
func RequireAuth(tokens *TokenManager) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, bearerPrefix) {
				httpx.WriteError(r.Context(), w, ErrUnauthorized)
				return
			}

			userID, err := tokens.Parse(strings.TrimPrefix(header, bearerPrefix))
			if err != nil {
				httpx.WriteError(r.Context(), w, ErrUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), userID)))
		})
	}
}
