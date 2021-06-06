package middleware

import (
	"context"
	"net/http"

	"asperitas-clone/pkg/session"

	"github.com/gorilla/mux"
)

type ReqTemplate struct {
	Method string
	Reg    string
}

type AuthService struct {
	SessionManager *session.SessionManager
	NeedAuth       []ReqTemplate
}

func (auth AuthService) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		if route == nil {
			next.ServeHTTP(w, r)
			return
		}
		methods, err := route.GetMethods()
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		template, err := mux.CurrentRoute(r).GetPathTemplate()
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		for _, req := range auth.NeedAuth {
			for _, method := range methods {
				if method == req.Method && template == req.Reg {
					sess, err := auth.SessionManager.Check(r)
					if err != nil {
						http.Error(w, `Can't get session`, http.StatusInternalServerError)
						return
					}
					if sess == nil {
						http.Redirect(w, r, "/", http.StatusUnauthorized)
						return
					}
					ctx := context.WithValue(r.Context(), session.SessionKey, sess)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
