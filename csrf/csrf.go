package csrf

import (
	"context"
	"net/http"

	"github.com/gorilla/securecookie"

	"app/session"
)

type key int

const tokenKey = key(1)
const signerKey = key(2)

const CookiesName = "x-csrf-token"

func GetToken(r *http.Request) string {
	return r.Context().Value(CookiesName).(string)
}

func New() func(http.Handler) http.Handler {
	// Hash keys should be at least 32 bytes long
	var hashKey = []byte("very-secret")
	// Block keys should be 16 bytes (AES-128) or 32 bytes (AES-256) long.
	// Shorter keys may weaken the encryption used.
	var blockKey = []byte("a-lot-secret")
	var signer = securecookie.New(hashKey, blockKey).SetSerializer(securecookie.JSONEncoder{})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := session.GetSession(r.Context())

			if s.UserID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			c, err := r.Cookie(CookiesName)
			if err == nil {
				next.ServeHTTP(w, r.WithContext(
					context.WithValue(context.WithValue(r.Context(), signerKey, signer), tokenKey, c.Value),
				))
			}

			encoded, err := signer.Encode(CookiesName, cookieValue{UserID: s.UserID})
			if err == nil {
				http.SetCookie(w, &http.Cookie{
					Name:     CookiesName,
					Value:    encoded,
					Path:     "/",
					Secure:   true,
					HttpOnly: true,
				})
			}

			next.ServeHTTP(w, r.WithContext(
				context.WithValue(context.WithValue(r.Context(), signerKey, signer), tokenKey, encoded),
			))
		})
	}
}

func Verify(r *http.Request, token string) bool {
	signer := r.Context().Value(signerKey).(*securecookie.SecureCookie)

	var v cookieValue
	err := signer.Decode(CookiesName, token, &v)
	if err != nil {
		return false
	}

	return v.UserID == session.GetSession(r.Context()).UserID
}

type cookieValue struct {
	UserID int32 `json:"user_id"`
}
