package users

import (
	"context"
	"errors"
	"net/http"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/middleware"
	"sakukota-go/internal/pkg/httpx"
	"sakukota-go/internal/storage"
)

// codeUserNotFound is sent when the token's user no longer exists.
const codeUserNotFound = "USER_NOT_FOUND"

//go:generate mockgen -source=handler.go -destination=mock/handler.go -package=mock

// Service is the service the Handler uses. Create and FetchByEmail are not
// called by the Handler; they are part of Service so Module.Service can be
// wired into auth.
type Service interface {
	Create(ctx context.Context, u *domain.User) (*domain.User, error)
	FetchByEmail(ctx context.Context, email string) (*domain.User, error)
	Fetch(ctx context.Context, id int64) (*domain.User, error)
	Rename(ctx context.Context, id int64, name string) (*domain.User, error)
	AvatarUploadURL(ctx context.Context, id int64, contentType string) (storage.Upload, error)
	SetAvatar(ctx context.Context, id int64, uploadKey string) (*domain.User, error)
}

// Handler serves the logged-in user's profile. Every route must sit behind
// middleware.WithAuth, so the user is known.
type Handler struct {
	service Service

	// mediaURL turns a stored media key into its public URL.
	mediaURL func(key string) string
}

// NewHandler returns a Handler that uses svc and builds media links with mediaURL.
func NewHandler(svc Service, mediaURL func(key string) string) *Handler {
	return &Handler{
		service:  svc,
		mediaURL: mediaURL,
	}
}

// Show sends back the logged-in user's profile.
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}

	u, err := h.service.Fetch(r.Context(), id)
	h.respond(w, u, err)
}

// Rename changes the logged-in user's display name and returns the profile.
func (h *Handler) Rename(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[RenameRequest](w, r)
	if !ok {
		return
	}

	u, err := h.service.Rename(r.Context(), id, req.Name)
	h.respond(w, u, err)
}

// CreateAvatarUpload returns a temporary link the logged-in user uploads a
// profile photo to directly.
func (h *Handler) CreateAvatarUpload(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.MediaUploadRequest](w, r)
	if !ok {
		return
	}

	upload, err := h.service.AvatarUploadURL(r.Context(), id, req.ContentType)
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, storage.ToMediaUploadResponse(upload))
}

// SetAvatar makes an already uploaded photo the logged-in user's profile
// photo and returns the profile.
func (h *Handler) SetAvatar(w http.ResponseWriter, r *http.Request) {
	id, ok := userID(w, r)
	if !ok {
		return
	}

	req, ok := httpx.DecodeJSON[storage.ConfirmUploadRequest](w, r)
	if !ok {
		return
	}

	u, err := h.service.SetAvatar(r.Context(), id, req.UploadKey)
	h.respond(w, u, err)
}

// respond answers with the profile u, or with err if it is not nil.
func (h *Handler) respond(w http.ResponseWriter, u *domain.User, err error) {
	if err != nil {
		respondError(w, err)
		return
	}

	httpx.RespondJSON(w, http.StatusOK, toResponse(u, h.mediaURL))
}

// respondError answers an error from a profile operation.
func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		httpx.RespondError(w, http.StatusNotFound, codeUserNotFound, err.Error())
	case errors.Is(err, domain.ErrUserNameRequired),
		errors.Is(err, domain.ErrUserNameTooLong),
		errors.Is(err, domain.ErrAvatarNotImage),
		storage.IsInvalidMedia(err):
		httpx.RespondError(w, http.StatusBadRequest, httpx.CodeValidationError, err.Error())
	default:
		httpx.RespondInternalError(w)
	}
}

// userID reads the logged-in user, or answers 401.
func userID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.RespondError(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing user")
	}
	return id, ok
}
