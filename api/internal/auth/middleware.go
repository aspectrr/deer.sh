package auth

import (
	"context"
	"net/http"

	"fmt"
	serverError "github.com/aspectrr/fluid.sh/api/internal/error"
	"github.com/aspectrr/fluid.sh/api/internal/store"
)

type userKey struct{}

func UserFromContext(ctx context.Context) *store.User {
	u, _ := ctx.Value(userKey{}).(*store.User)
	return u
}

// RequireAuth is middleware that validates session cookie and loads user into context.
func RequireAuth(st store.Store, secureCookies bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				serverError.RespondError(w, http.StatusUnauthorized, fmt.Errorf("authentication required"))
				return
			}

			sess, err := st.GetSession(r.Context(), HashSessionToken(cookie.Value))
			if err != nil {
				ClearSessionCookie(w, secureCookies)
				serverError.RespondError(w, http.StatusUnauthorized, fmt.Errorf("invalid or expired session"))
				return
			}

			user, err := st.GetUser(r.Context(), sess.UserID)
			if err != nil {
				ClearSessionCookie(w, secureCookies)
				serverError.RespondError(w, http.StatusUnauthorized, fmt.Errorf("user not found"))
				return
			}

			ctx := context.WithValue(r.Context(), userKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
