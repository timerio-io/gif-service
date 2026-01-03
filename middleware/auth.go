package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
)

type contextKey string

const UserIDKey contextKey = "userID"

type VerifyResponse struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authURL := os.Getenv("AUTH_WORKER_URL")
		apiKey := os.Getenv("GIF_SERVICE_API_KEY")

		req, err := http.NewRequest("GET", authURL+"/api/verify", nil)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		req.Header.Set("x-api-key", apiKey)

		if cookie, err := r.Cookie("better-auth.session_token"); err == nil {
			req.AddCookie(cookie)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Auth service unavailable", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var verifyResp VerifyResponse
		if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
			http.Error(w, "Invalid response", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, verifyResp.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
