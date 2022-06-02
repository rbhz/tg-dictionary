package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/rbhz/tg-dictionary/app/db"
	"github.com/rs/zerolog/log"
)

// JWTClaims custom claims with user id
type JWTClaims struct {
	User *int64 `json:"user"`
	jwt.StandardClaims
}

// AuthResponse response for authentication
type AuthResponse struct {
	Token string `json:"token"`
}

// authService implements methods for API authentication
type authService struct {
	telegramToken string
	jwtSecret     []byte
}

// createToken creates JWT token
func (s *authService) createToken(userID int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		User: &userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().UTC().Add(time.Hour * 24).Unix(),
			NotBefore: time.Now().UTC().Unix(),
		},
	})
	tokenStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return tokenStr, nil
}

// TelegramRedirectHandler handles authentication after Telegram redirect
func (s *authService) TelegramRedirectHandler(w http.ResponseWriter, r *http.Request) {
	// validate request
	secretKey := sha256.Sum256([]byte(s.telegramToken))
	keys := []string{}
	query := r.URL.Query()
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var data bytes.Buffer
	for _, key := range keys {
		if key == "hash" {
			continue
		}
		values := query[key]
		for _, val := range values {
			data.Write([]byte(fmt.Sprintf("%s=%s", key, val)))
		}
	}
	h := hmac.New(sha256.New, secretKey[:])
	h.Write(data.Bytes())
	hash := hex.EncodeToString(h.Sum(nil))
	if hash != r.URL.Query().Get("hash") {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
		return
	}
	userID, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		log.Error().Err(err).Str("userID", r.URL.Query().Get("id")).Msg("failed to parse user id")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid ID"))
		return
	}

	// create JWT token
	token, err := s.createToken(userID)
	if err != nil {
		log.Error().Err(err).Msg("failed to create token")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	jdata, jerr := json.Marshal(AuthResponse{Token: token})
	if jerr != nil {
		log.Error().Err(jerr).Msg("failed to marshal json")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(jdata)
}

// UserCtx checks authorization token and adds user to context
func (s *authService) UserCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestToken := r.Header.Get("Authorization")
		if !strings.HasPrefix(requestToken, "Bearer ") {
			requestToken = ""
		}
		requestToken = strings.Replace(requestToken, "Bearer ", "", 1)
		if requestToken == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		token, err := jwt.ParseWithClaims(requestToken, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}

		claims := token.Claims.(*JWTClaims)
		if claims.User == nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		now := time.Now().Unix()
		if claims.NotBefore > now || claims.ExpiresAt < now {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserIDKey, db.UserID(*claims.User))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
