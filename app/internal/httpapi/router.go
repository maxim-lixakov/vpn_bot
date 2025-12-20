package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"vpn-app/internal/config"
	"vpn-app/internal/domain"
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

	clients map[string]*outline.Client // country -> outline client
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

// ---------- Telegram: upsert user + return state/subscription ----------

type tgUpsertReq struct {
	TgUserID     int64   `json:"tg_user_id"`
	Username     *string `json:"username"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	LanguageCode *string `json:"language_code"`
	Phone        *string `json:"phone"`
}

type tgUpsertResp struct {
	UserID          int64      `json:"user_id"`
	State           string     `json:"state"`
	SelectedCountry *string    `json:"selected_country"`
	SubscriptionOK  bool       `json:"subscription_ok"`
	ActiveUntil     *time.Time `json:"active_until"`
}

func (s *Server) handleTelegramUpsert(w http.ResponseWriter, r *http.Request) {
	var req tgUpsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	u := repo.User{
		TgUserID: req.TgUserID,
	}
	if req.Username != nil {
		u.Username = sql.NullString{String: *req.Username, Valid: true}
	}
	if req.FirstName != nil {
		u.FirstName = sql.NullString{String: *req.FirstName, Valid: true}
	}
	if req.LastName != nil {
		u.LastName = sql.NullString{String: *req.LastName, Valid: true}
	}
	if req.LanguageCode != nil {
		u.LanguageCode = sql.NullString{String: *req.LanguageCode, Valid: true}
	}
	if req.Phone != nil {
		u.Phone = sql.NullString{String: *req.Phone, Valid: true}
	}

	user, err := s.users.UpsertByTelegram(r.Context(), u)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	st, err := s.states.EnsureDefault(r.Context(), user.ID, domain.StateChooseCountry)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	now := time.Now().UTC()
	until, ok, err := s.subs.GetActiveUntil(r.Context(), user.ID, now)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	var sel *string
	if st.SelectedCountry.Valid {
		sel = &st.SelectedCountry.String
	}
	var au *time.Time
	if !until.IsZero() {
		au = &until
	}

	writeJSON(w, tgUpsertResp{
		UserID:          user.ID,
		State:           st.State,
		SelectedCountry: sel,
		SubscriptionOK:  ok,
		ActiveUntil:     au,
	})
}

// ---------- Telegram: set state ----------

type tgSetStateReq struct {
	TgUserID        int64   `json:"tg_user_id"`
	State           string  `json:"state"`
	SelectedCountry *string `json:"selected_country"`
}

type tgSetStateResp struct {
	State           string  `json:"state"`
	SelectedCountry *string `json:"selected_country"`
}

func (s *Server) handleTelegramSetState(w http.ResponseWriter, r *http.Request) {
	var req tgSetStateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 || req.State == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, ok, err := s.users.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil || !ok {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	var sel sql.NullString
	if req.SelectedCountry != nil {
		sel = sql.NullString{String: *req.SelectedCountry, Valid: true}
	}

	st, err := s.states.Set(r.Context(), user.ID, req.State, sel)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	var outSel *string
	if st.SelectedCountry.Valid {
		outSel = &st.SelectedCountry.String
	}

	writeJSON(w, tgSetStateResp{State: st.State, SelectedCountry: outSel})
}

// ---------- Telegram: mark paid subscription ----------

type tgMarkPaidReq struct {
	TgUserID                int64  `json:"tg_user_id"`
	AmountMinor             int64  `json:"amount_minor"`
	Currency                string `json:"currency"`
	TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
	ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
}

type tgMarkPaidResp struct {
	ActiveUntil time.Time `json:"active_until"`
}

func (s *Server) handleTelegramMarkPaid(w http.ResponseWriter, r *http.Request) {
	var req tgMarkPaidReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 || req.Currency == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	user, ok, err := s.users.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil || !ok {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	until, err := s.subs.MarkPaid(r.Context(), repo.MarkPaidArgs{
		UserID:                  user.ID,
		Provider:                "telegram",
		AmountMinor:             req.AmountMinor,
		Currency:                req.Currency,
		TelegramPaymentChargeID: sql.NullString{String: req.TelegramPaymentChargeID, Valid: req.TelegramPaymentChargeID != ""},
		ProviderPaymentChargeID: sql.NullString{String: req.ProviderPaymentChargeID, Valid: req.ProviderPaymentChargeID != ""},
		PaidAt:                  time.Now().UTC(),
	})
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	// move state to ISSUE_KEY (selected_country remains as-is)
	st, err := s.states.Get(r.Context(), user.ID)
	if err == nil {
		_ = func() error {
			_, err := s.states.Set(r.Context(), user.ID, domain.StateIssueKey, st.SelectedCountry)
			return err
		}()
	}

	writeJSON(w, tgMarkPaidResp{ActiveUntil: until})
}

// ---------- Issue key (requires subscription active) ----------

type issueKeyReq struct {
	TgUserID int64  `json:"tg_user_id"`
	Country  string `json:"country"`
}

type issueKeyResp struct {
	ServerName  string `json:"server_name"`
	Country     string `json:"country"`
	AccessKeyID string `json:"access_key_id"`
	AccessURL   string `json:"access_url"`
}

func (s *Server) handleIssueKey(w http.ResponseWriter, r *http.Request) {
	var req issueKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TgUserID == 0 || req.Country == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Country = strings.ToLower(req.Country)

	server, ok := s.cfg.Servers[req.Country]
	if !ok {
		http.Error(w, "unknown country", http.StatusBadRequest)
		return
	}

	user, okU, err := s.users.GetByTelegramID(r.Context(), req.TgUserID)
	if err != nil || !okU {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	_, subOK, err := s.subs.GetActiveUntil(r.Context(), user.ID, now)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}
	if !subOK {
		http.Error(w, "subscription inactive", http.StatusPaymentRequired)
		return
	}

	// if already have active key
	if k, ok, err := s.keys.GetActive(r.Context(), user.ID, req.Country); err == nil && ok {
		_ = func() error {
			_, err := s.states.Set(r.Context(), user.ID, domain.StateActive, sql.NullString{String: req.Country, Valid: true})
			return err
		}()
		writeJSON(w, issueKeyResp{
			ServerName:  server.Name,
			Country:     req.Country,
			AccessKeyID: k.OutlineKeyID,
			AccessURL:   k.AccessURL,
		})
		return
	}

	client := s.clients[req.Country]
	key, err := client.CreateAccessKey(r.Context(), "tg:"+itoa(req.TgUserID))
	if err != nil {
		http.Error(w, "outline create key failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	if err := s.keys.Insert(r.Context(), user.ID, req.Country, key.ID, key.AccessURL); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusBadGateway)
		return
	}

	_, _ = s.states.Set(r.Context(), user.ID, domain.StateActive, sql.NullString{String: req.Country, Valid: true})

	writeJSON(w, issueKeyResp{
		ServerName:  server.Name,
		Country:     req.Country,
		AccessKeyID: key.ID,
		AccessURL:   key.AccessURL,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func itoa(x int64) string {
	if x == 0 {
		return "0"
	}
	neg := x < 0
	if neg {
		x = -x
	}
	var b [32]byte
	i := len(b)
	for x > 0 {
		i--
		b[i] = byte('0' + (x % 10))
		x /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
