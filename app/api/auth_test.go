package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectTelegramHandler(t *testing.T) {
	const path = "/api/v1/auth/telegram"
	requestParams := map[string]string{
		"id":         "1",
		"first_name": "John",
		"username":   "jDoe",
		"photo_url":  "http://test.com/image.png",
		"auth_date":  "1653049612",
		"hash":       "7403f73546fca556ae1a79942e7ea43d593052fde2ee71f122785c4cbaf53c27",
	}
	paramsOrder := []string{"id", "first_name", "username", "photo_url", "auth_date", "hash"}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	checkValidToken := func(t *testing.T, token string) {
		parsedToken, err := jwt.ParseWithClaims(token, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(testJWTSecret), nil
		})
		require.NoError(t, err)
		claims, ok := parsedToken.Claims.(*JWTClaims)
		require.True(t, ok)
		expected := int64(1)
		assert.Equal(t, &expected, claims.User)
		assert.Less(t, time.Now().Unix(), claims.ExpiresAt)
		assert.GreaterOrEqual(t, time.Now().Unix(), claims.NotBefore)
	}

	t.Run("success", func(t *testing.T) {
		ts, cancel := getTestServer(nil)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		require.NoError(t, err)
		q := req.URL.Query()
		for _, p := range paramsOrder {
			q.Add(p, requestParams[p])
		}
		req.URL.RawQuery = q.Encode()
		r, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, r.StatusCode)

		var rData AuthResponse
		err = json.NewDecoder(r.Body).Decode(&rData)
		require.NoError(t, err)
		assert.NotEmpty(t, rData.Token)
		checkValidToken(t, rData.Token)
	})
	t.Run("success reverse params order", func(t *testing.T) {
		ts, cancel := getTestServer(nil)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		require.NoError(t, err)
		q := req.URL.Query()
		for idx := len(paramsOrder) - 1; idx >= 0; idx-- {
			p := paramsOrder[idx]
			q.Add(p, requestParams[p])
		}
		req.URL.RawQuery = q.Encode()
		r, err := client.Do(req)
		assert.NoError(t, err)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, r.StatusCode)

		var rData AuthResponse
		err = json.NewDecoder(r.Body).Decode(&rData)
		require.NoError(t, err)
		assert.NotEmpty(t, rData.Token)
		checkValidToken(t, rData.Token)
	})
	t.Run("invalid hash", func(t *testing.T) {
		ts, cancel := getTestServer(nil)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		require.NoError(t, err)
		q := req.URL.Query()
		for _, p := range paramsOrder {
			if p == "hash" {
				q.Add(p, "invalid")
			} else {
				q.Add(p, requestParams[p])
			}
		}
		req.URL.RawQuery = q.Encode()
		r, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", string(body))
	})
	t.Run("missing id", func(t *testing.T) {
		ts, cancel := getTestServer(nil)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		require.NoError(t, err)
		q := req.URL.Query()
		for _, p := range paramsOrder {
			if p != "id" {
				q.Add(p, requestParams[p])
			}
		}
		req.URL.RawQuery = q.Encode()
		r, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", string(body))

	})
	t.Run("invalid id", func(t *testing.T) {
		ts, cancel := getTestServer(nil)
		defer cancel()
		req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		require.NoError(t, err)
		q := req.URL.Query()
		for _, p := range paramsOrder {
			if p == "id" {
				q.Add(p, "invalid")
			} else {
				q.Add(p, requestParams[p])
			}
		}
		req.URL.RawQuery = q.Encode()
		r, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", string(body))
	})
}

func TestUserCtxMiddleware(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}
	s := &authService{telegramToken: testTGToken, jwtSecret: []byte(testJWTSecret)}
	handler := s.UserCtx(&emptyHandler{})

	checkSuccess := func(t *testing.T, header string) {
		for _, method := range methods {
			req, err := http.NewRequest(method, "/", nil)
			require.NoError(t, err)
			req.Header.Add("Authorization", header)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			r := recorder.Result()
			assert.Equal(t, http.StatusOK, r.StatusCode)
		}
	}
	checkError := func(t *testing.T, header string) {
		for _, method := range methods {
			req, err := http.NewRequest(method, "/", nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			r := recorder.Result()
			assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Equal(t, "unauthorized", string(body))
		}
	}
	t.Run("success", func(t *testing.T) {
		testJWT, err := s.createToken(1)
		require.NoError(t, err)
		checkSuccess(t, "Bearer "+testJWT)
	})
	t.Run("without header", func(t *testing.T) {
		checkError(t, "")
	})
	t.Run("invalid JWT", func(t *testing.T) {
		checkError(t, "Bearer invalidJWT")
	})
	t.Run("invalid prefix", func(t *testing.T) {
		testJWT, err := s.createToken(1)
		require.NoError(t, err)
		checkError(t, "Invalid "+testJWT)
	})
	t.Run("invalid JWT sign", func(t *testing.T) {
		testJWT, err := (&authService{telegramToken: testTGToken, jwtSecret: []byte(testJWTSecret + "1")}).createToken(1)
		require.NoError(t, err)
		checkError(t, "Bearer "+testJWT)
	})
	t.Run("empty user", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
			User: nil,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().UTC().Add(time.Hour * 24).Unix(),
				NotBefore: time.Now().UTC().Unix(),
			},
		})
		testJWT, err := token.SignedString(s.jwtSecret)
		require.NoError(t, err)
		checkError(t, "Bearer "+testJWT)

	})
	t.Run("invalid JWT claims", func(t *testing.T) {
		type invalidJWTClaims struct {
			User string `json:"user"`
			jwt.StandardClaims
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, invalidJWTClaims{
			User: "1",
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().UTC().Add(time.Hour * 24).Unix(),
				NotBefore: time.Now().UTC().Unix(),
			},
		})
		testJWT, err := token.SignedString(s.jwtSecret)
		require.NoError(t, err)
		checkError(t, "Bearer "+testJWT)
	})
	t.Run("expired JWT", func(t *testing.T) {
		userID := int64(1)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
			User: &userID,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().UTC().Add(-1 * time.Hour).Unix(),
				NotBefore: time.Now().UTC().Unix(),
			},
		})
		testJWT, err := token.SignedString(s.jwtSecret)
		require.NoError(t, err)
		checkError(t, "Bearer "+testJWT)
	})
	t.Run("invalid before", func(t *testing.T) {
		userID := int64(1)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
			User: &userID,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().UTC().Add(time.Hour * 25).Unix(),
				NotBefore: time.Now().UTC().Add(time.Hour * 1).Unix(),
			},
		})
		testJWT, err := token.SignedString(s.jwtSecret)
		require.NoError(t, err)
		checkError(t, "Bearer "+testJWT)
	})
}
