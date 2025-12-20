package handlers

import (
	"database/sql"
	"github.com/go-chi/chi/v5"
	"net/http"

	"vpn-app/internal/config"
	"vpn-app/internal/outline"
	"vpn-app/internal/repo"
)

type Server struct {
	cfg config.Config
	db  *sql.DB

	users  *repo.UsersRepo
	states *repo.StateRepo
	subs   *repo.SubscriptionsRepo
	keys   *repo.AccessKeysRepo

	clients map[string]*outline.Client
}

func New(cfg config.Config, db *sql.DB) *Server {
	clients := make(map[string]*outline.Client, len(cfg.Servers))
	for code, s := range cfg.Servers {
		clients[code] = outline.NewClient(s.APIURL, s.TLSInsecure)
	}

	return &Server{
		cfg:     cfg,
		db:      db,
		users:   repo.NewUsersRepo(db),
		states:  repo.NewStateRepo(db),
		subs:    repo.NewSubscriptionsRepo(db),
		keys:    repo.NewAccessKeysRepo(db),
		clients: clients,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// bot-only endpoints
	r.Group(func(r chi.Router) {
		r.Use(s.auth)

		r.Post("/v1/telegram/upsert", s.handleTelegramUpsert)
		r.Post("/v1/telegram/set-state", s.handleTelegramSetState)
		r.Post("/v1/telegram/mark-paid", s.handleTelegramMarkPaid)

		r.Post("/v1/issue-key", s.handleIssueKey)
	})

	return r
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Token") != s.cfg.InternalToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
