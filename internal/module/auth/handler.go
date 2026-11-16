package auth

import (
	"errors"
	"net/http"

	"rantau-go/internal/config"
	"rantau-go/internal/pkg/httpx"
)

type Handler struct {
	cfg     config.Config
	service Service
}

// NewHandler creates a Handler that reads cookie settings from cfg and passes work to the service.
func NewHandler(cfg config.Config, service Service) *Handler {
	return &Handler{
		cfg:     cfg,
		service: service,
	}
}

// -----------------------------------------------------------------------------
// Handler
// -----------------------------------------------------------------------------

// Register reads the sign-up body and creates the user.
// It sends back the new user, or a validation or email-taken error.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	req, ok := httpx.DecodeJSON[RegisterRequest](w, r)
	if !ok {
		return
	}

	u, err := h.service.Register(r.Context(), toUser(*req), req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidName),
			errors.Is(err, ErrNameTooLong),
			errors.Is(err, ErrInvalidEmail),
			errors.Is(err, ErrEmailTooLong),
			errors.Is(err, ErrWeakPassword),
			errors.Is(err, ErrPasswordTooLong):
			httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
		case errors.Is(err, ErrEmailTaken):
			httpx.RespondError(w, http.StatusConflict, httpx.CodeEmailTaken, err.Error())
		default:
			httpx.RespondInternalError(w)
		}
		return
	}

	httpx.RespondJSON(w, http.StatusCreated, toUserResponse(u))
}

// Login checks the email and password, then sends back the user and an access token.
// The refresh token goes in a cookie, not in the body.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	req, ok := httpx.DecodeJSON[LoginRequest](w, r)
	if !ok {
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

// Refresh trades the refresh token in the cookie for a new session.
// If the token is missing, used, or expired, the cookie is cleared and it sends 401.
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

// Logout revokes the refresh token in the cookie and clears the cookie.
// The cookie is cleared even if revoking fails.
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

// setRefreshCookie puts token in the refresh cookie, lasting as long as refresh tokens do.
func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	h.writeRefreshCookie(w, token, int(h.cfg.RefreshTokenExpiry.Seconds()))
}

// clearRefreshCookie expires the refresh cookie, so the browser deletes it.
func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	h.writeRefreshCookie(w, "", -1)
}

// writeRefreshCookie sets the refresh cookie to token with the given max age.
// The cookie is HttpOnly, so page scripts cannot read it, and SameSite=Strict.
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
