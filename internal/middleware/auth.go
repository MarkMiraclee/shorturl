package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userID"

var secretKey = []byte("super-secret-key-that-is-not-so-secret")

func sign(data string) string {
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID string
		newCookie := false

		cookie, err := r.Cookie("user_id")
		if err != nil {
			newCookie = true
		} else {
			parts := strings.Split(cookie.Value, "|")
			if len(parts) == 2 && sign(parts[0]) == parts[1] {
				userID = parts[0]
			} else {
				newCookie = true
			}
		}

		if newCookie {
			userID = uuid.NewString()
			signedUserID := userID + "|" + sign(userID)
			http.SetCookie(w, &http.Cookie{
				Name:  "user_id",
				Value: signedUserID,
				Path:  "/",
			})
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
