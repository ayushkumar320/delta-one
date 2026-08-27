package middleware

import (
	"context"
	"net/http"

	"github.com/ayush/delta-one/shared/httpx"
)

// Headers the gateway sets after it verifies an access token. Services behind
// the gateway trust them; they are never accepted from the public internet
// because the gateway strips them from inbound requests.
const (
	HeaderUserID    = "X-User-ID"
	HeaderUserEmail = "X-User-Email"
	HeaderRole      = "X-User-Role"
)

const (
	userIDKey ctxKey = "user_id"
	emailKey  ctxKey = "user_email"
	roleKey   ctxKey = "user_role"
)

// UserIDFrom returns the caller's user ID, or "" for an anonymous request.
func UserIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

// EmailFrom returns the caller's email address, or "" for an anonymous
// request. Services use it to address notifications without calling auth.
func EmailFrom(ctx context.Context) string {
	email, _ := ctx.Value(emailKey).(string)
	return email
}

// RoleFrom returns the caller's role, or "" for an anonymous request.
func RoleFrom(ctx context.Context) string {
	role, _ := ctx.Value(roleKey).(string)
	return role
}

// Identity reads the identity headers set by the gateway into the context. It
// does not reject anonymous requests; handlers that need a user use RequireUser.
func Identity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if id := r.Header.Get(HeaderUserID); id != "" {
			ctx = context.WithValue(ctx, userIDKey, id)
			ctx = context.WithValue(ctx, emailKey, r.Header.Get(HeaderUserEmail))
			ctx = context.WithValue(ctx, roleKey, r.Header.Get(HeaderRole))
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireUser rejects requests that carry no identity.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserIDFrom(r.Context()) == "" {
			httpx.Fail(w, httpx.Unauthorized("unauthenticated", "sign in to continue"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole rejects requests whose caller does not hold the given role.
func RequireRole(role string) Middleware {
	return func(next http.Handler) http.Handler {
		return RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if RoleFrom(r.Context()) != role {
				httpx.Fail(w, httpx.Forbidden("forbidden", "you do not have access to this resource"))
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}
