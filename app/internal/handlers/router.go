package handlers

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"

	"vpn-app/internal/config"
	"vpn-app/internal/outline"
	"vpn-app/internal/repo"
)

type Server struct {
	cfg config.Config
	db  *sql.DB

	usersRepo           repo.UsersRepoInterface
	statesRepo          repo.StateRepoInterface
	subsRepo            repo.SubscriptionsRepoInterface
	keysRepo            repo.AccessKeysRepoInterface
	countriesAddRepo    repo.CountriesToAddRepoInterface
	promocodesRepo      repo.PromocodesRepoInterface
	promocodeUsagesRepo repo.PromocodeUsagesRepoInterface
	feedbackRepo        repo.FeedbackRepoInterface

	clients map[string]outline.OutlineClientInterface
}

func New(cfg config.Config, db *sql.DB) *Server {
	clients := make(map[string]outline.OutlineClientInterface, len(cfg.Servers))
	for code, s := range cfg.Servers {
		clients[code] = outline.NewClient(s.APIURL, s.TLSInsecure)
	}

	return &Server{
		cfg:                 cfg,
		db:                  db,
		usersRepo:           repo.NewUsersRepo(db),
		statesRepo:          repo.NewStateRepo(db),
		subsRepo:            repo.NewSubscriptionsRepo(db),
		keysRepo:            repo.NewAccessKeysRepo(db),
		countriesAddRepo:    repo.NewCountriesToAddRepo(db),
		promocodesRepo:      repo.NewPromocodesRepo(db),
		promocodeUsagesRepo: repo.NewPromocodeUsagesRepo(db),
		feedbackRepo:        repo.NewFeedbackRepo(db),
		clients:             clients,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Group(func(r chi.Router) {
		r.Use(s.auth)

		r.Post("/v1/telegram/upsert", s.handleTelegramUpsert)
		r.Post("/v1/telegram/set-state", s.handleTelegramSetState)
		r.Post("/v1/telegram/mark-paid", s.handleTelegramMarkPaid)
		r.Get("/v1/telegram/subscriptions", s.handleTelegramSubscriptions)
		r.Get("/v1/telegram/country-status", s.handleTelegramCountryStatus)
		r.Post("/v1/telegram/countries-to-add", s.handleTelegramCountriesToAdd)
		r.Post("/v1/telegram/promocode-use", s.handleTelegramPromocodeUse)
		r.Post("/v1/telegram/promocode-rollback", s.handleTelegramPromocodeRollback)
		r.Post("/v1/telegram/update-promocode-subscription", s.handleTelegramUpdatePromocodeSubscription)
		r.Post("/v1/telegram/feedback", s.handleTelegramFeedback)
		r.Post("/v1/telegram/referral-code", s.handleTelegramReferralCode)

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
