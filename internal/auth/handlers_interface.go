package auth

import "net/http"

// AuthHandlers is the set of HTTP handlers exposed by the auth domain.
// Unlike the other domains, auth has no RegisterRoutes: Register and Login
// are individually wrapped with their own rate limiters in main, so each
// method is wired by hand there instead of through one shared method.
type AuthHandlers interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	Refresh(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	Me(w http.ResponseWriter, r *http.Request)
}
