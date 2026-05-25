package main

import (
	"log/slog"
	"net/http"
)

func (app *application) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recover",
					slog.Any("error", err),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
				)
				app.writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
