package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rbhz/tg-dictionary/app/db"
)

type ctxKey int

const (
	ctxUserIDKey ctxKey = iota
)

// Server is the API main server struct
type Server struct {
	storage db.Storage
	router  chi.Router
}

// Run starts the server
func (s *Server) Run(port int) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), s.router)
}

func (s *Server) setJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// NewServer creates a new server
func NewServer(storage db.Storage, tgToken string, jwtSecret string) *Server {
	s := &Server{storage: storage}
	dict := dictionaryService{storage: storage}
	auth := authService{telegramToken: tgToken, jwtSecret: []byte(jwtSecret)}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.setJSONContentType)
		r.Route("/auth", func(r chi.Router) {
			r.Get("/telegram", auth.TelegramRedirectHandler)
		})
		r.Route("/dictionary", func(r chi.Router) {
			r.Use(auth.UserCtx)
			r.Get("/", dict.GetUserDictionary)
			r.Get("/word/{word}", dict.GetWord)
			r.Post("/word/{word}", dict.UpdateWord)
		})

	})

	s.router = r
	return s
}
