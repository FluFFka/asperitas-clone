package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type ReqLogger struct {
	Logger *zap.SugaredLogger
}

func (req ReqLogger) AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		req.Logger.Infow("New request",
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
			"url", r.URL.Path,
			"time", time.Since(start),
		)
	})
}
