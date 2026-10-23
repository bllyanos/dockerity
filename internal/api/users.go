package api

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/bllyanos/dockerity/internal/auth"
	"github.com/bllyanos/dockerity/internal/domain"
)

// UserAdminService manages users. Every route using it is admin only.
type UserAdminService interface {
	List(ctx context.Context) ([]domain.User, error)
	Create(ctx context.Context, username, password string, isAdmin bool) (domain.User, error)
	SetDisabled(ctx context.Context, actor domain.User, id int64, disabled bool) (domain.User, error)
	ResetPassword(ctx context.Context, id int64, password string) (domain.User, error)
}

type userHandlers struct {
	logger *slog.Logger
	users  UserAdminService
	audit  auditor
}

func (h userHandlers) register(rt *routes) {
	rt.handle("GET /users", adminOnly(h.list))
	rt.handle("POST /users", adminOnly(h.create))
	rt.handle("POST /users/{id}/disable", adminOnly(h.setDisabled(true)))
	rt.handle("POST /users/{id}/enable", adminOnly(h.setDisabled(false)))
	rt.handle("POST /users/{id}/password", adminOnly(h.resetPassword))
}

func (h userHandlers) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	views := make([]userView, 0, len(users))
	for _, u := range users {
		views = append(views, toUserView(u))
	}
	writeJSON(w, http.StatusOK, map[string][]userView{"users": views})
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"isAdmin"`
}

func (h userHandlers) create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	u, err := h.users.Create(r.Context(), req.Username, req.Password, req.IsAdmin)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, userEvent(domain.AuditUserCreated, u))
	writeJSON(w, http.StatusCreated, map[string]userView{"user": toUserView(u)})
}

func (h userHandlers) setDisabled(disabled bool) http.HandlerFunc {
	action := domain.AuditUserEnabled
	if disabled {
		action = domain.AuditUserDisabled
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		p, _ := auth.PrincipalFrom(r.Context())
		u, err := h.users.SetDisabled(r.Context(), p.User, id, disabled)
		if err != nil {
			writeServiceError(w, r, h.logger, err)
			return
		}
		h.audit.record(r, userEvent(action, u))
		writeJSON(w, http.StatusOK, map[string]userView{"user": toUserView(u)})
	}
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func (h userHandlers) resetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req resetPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	u, err := h.users.ResetPassword(r.Context(), id, req.Password)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	h.audit.record(r, userEvent(domain.AuditUserPasswordReset, u))
	w.WriteHeader(http.StatusNoContent)
}

func userEvent(action string, u domain.User) domain.AuditEvent {
	tt, tid := userTarget(u)
	return domain.AuditEvent{Action: action, TargetType: tt, TargetID: tid}
}

// pathID parses the {id} path value, writing a 400 when it is not an integer.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id: must be an integer")
		return 0, false
	}
	return id, true
}
