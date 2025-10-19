package csrf

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/rs/zerolog/log"

	"app/session"
)

type key int

const tokenKey = key(1)
const signerKey = key(2)

const CookiesName = "x-csrf-token-3"
const FormName = "x-csrf-token"

func GetToken(r *http.Request) string {
	v := r.Context().Value(tokenKey)
	if v == nil {
		return ""
	}

	return v.(string)
}

func New() func(http.Handler) http.Handler {
	// Hash keys should be at least 32 bytes long
	var hashKey = []byte("very-secret1234")
	// Block keys should be 16 bytes (AES-128) or 32 bytes (AES-256) long.
	// Shorter keys may weaken the encryption used.
	var blockKey = []byte("a-lot-secret1234")
	var signer = securecookie.New(hashKey, blockKey).SetSerializer(securecookie.JSONEncoder{})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := session.GetSession(r.Context())

			if s.UserID == 0 {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, signerKey, signer)

			c, err := r.Cookie(CookiesName)
			if err == nil && c.Value != "" {
				ctx = context.WithValue(ctx, tokenKey, c.Value)

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			encoded, err := signer.Encode(CookiesName, cookieValue{UserID: s.UserID})
			if err != nil {
				log.Err(err).Msg("failed to generate csrf token")
				next.ServeHTTP(w, r)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     CookiesName,
				Value:    encoded,
				Path:     "/",
				Secure:   true,
				HttpOnly: true,
				MaxAge:   int(time.Hour/time.Second) * 24 * 7,
			})

			ctx = context.WithValue(ctx, tokenKey, encoded)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newToken(ctx context.Context, signer *securecookie.SecureCookie) (string, error) {
	encoded, err := signer.Encode(CookiesName, cookieValue{UserID: session.GetSession(ctx).UserID})
	if err != nil {
		return "", err
	}

	return encoded, nil
}

func Verify(r *http.Request, formValue string) bool {
	signer := r.Context().Value(signerKey).(*securecookie.SecureCookie)
	cookieToken := r.Context().Value(tokenKey).(string)

	if cookieToken != formValue {
		return false
	}

	var v cookieValue
	err := signer.Decode(CookiesName, formValue, &v)
	if err != nil {
		return false
	}

	return v.UserID == session.GetSession(r.Context()).UserID
}

func Clear(w http.ResponseWriter, r *http.Request) {
	token, err := newToken(r.Context(), r.Context().Value(signerKey).(*securecookie.SecureCookie))
	if err != nil {
		panic("failed to encode new token")
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookiesName,
		Value:    token,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		MaxAge:   int(time.Hour/time.Second) * 24 * 7,
	})
}

type cookieValue struct {
	UserID int32 `json:"user_id"`
}
