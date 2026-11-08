package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"rantau-go/internal/config"
	"rantau-go/internal/module/users"
	"rantau-go/internal/pkg/httpx"
)

type Handler struct {
	cfg     config.Config
	service Service
}

func NewHandler(cfg config.Config, service Service) *Handler {
	return &Handler{
		cfg:     cfg,
		service: service,
	}
}

// -----------------------------------------------------------------------------
// Handler
// -----------------------------------------------------------------------------

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req *RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req == nil {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidBody, "invalid request body")
		return
	}

	u, err := h.service.Register(r.Context(), toUser(*req), req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidName),
			errors.Is(err, ErrNameTooLong),
			errors.Is(err, ErrInvalidEmail),
			errors.Is(err, ErrEmailTooLong),
			errors.Is(err, ErrWeakPassword):
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
		case errors.Is(err, users.ErrEmailTaken):
			httpx.RespondError(w, http.StatusConflict, httpx.CodeEmailTaken, err.Error())
		default:
			httpx.RespondInternalError(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toUserResponse(u))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req *LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req == nil {
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeInvalidBody, "invalid request body")
		return
	}

	u, tokens, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeInvalidCredentials, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	h.setRefreshCookie(w, tokens.RefreshToken)
	httpx.RespondJSON(w, http.StatusOK, toSessionResponse(u, tokens))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		h.clearRefreshCookie(w)
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeInvalidRefreshToken, ErrInvalidRefreshToken.Error())
		return
	}

	u, tokens, err := h.service.Refresh(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			h.clearRefreshCookie(w)
			httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeInvalidRefreshToken, err.Error())
			return
		}
		httpx.RespondInternalError(w)
		return
	}

	h.setRefreshCookie(w, tokens.RefreshToken)
	httpx.RespondJSON(w, http.StatusOK, toSessionResponse(u, tokens))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// The browser drops the cookie even if revoking it server-side fails.
	h.clearRefreshCookie(w)

	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		if err := h.service.Logout(r.Context(), cookie.Value); err != nil {
			httpx.RespondInternalError(w)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// Refresh cookie
// -----------------------------------------------------------------------------

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	h.writeRefreshCookie(w, token, int(h.cfg.RefreshTokenExpiry.Seconds()))
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	h.writeRefreshCookie(w, "", -1)
}

func (h *Handler) writeRefreshCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}
