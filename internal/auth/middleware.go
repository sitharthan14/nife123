package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/nifetency/nife.io/internal/users"
	"github.com/nifetency/nife.io/pkg/jwt"
)

var userCtxKey = &contextKey{"user"}

type contextKey struct {
	email string
}

func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			headers := strings.Fields(authHeader)
			// Allow unauthenticated users in
			if len(headers) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			//validate jwt token
			tokenStr := headers[1]
			if len(tokenStr) == 0 {
				http.Error(w, "Empty token", http.StatusForbidden)
				return
			}
			email, stripeProductId, firstName, lastName, comapanyName, roleId, customerStripeId,err := jwt.ParseToken(tokenStr)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// create user and check if user exists in db
			user := users.User{Email: email, StripeProductId: stripeProductId, FirstName: firstName, LastName: lastName, CompanyName: comapanyName, RoleId: roleId,CustomerStripeId: customerStripeId}
			checkEmail := jwt.IsEmailValid(email)
			var ID int
			if checkEmail {
				id, err := users.GetUserIdByEmail(email)
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
				ID = id
			}
			user.ID = strconv.Itoa(ID)
			// put it in context
			ctx := context.WithValue(r.Context(), userCtxKey, &user)

			// and call the next with our new context
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) *users.User {
	raw, _ := ctx.Value(userCtxKey).(*users.User)
	return raw
}
