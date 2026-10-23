package api

import (
	"time"

	"github.com/bllyanos/dockerity/internal/domain"
)

// userView is the public JSON shape of a user. It never includes the password hash.
type userView struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	IsAdmin   bool      `json:"isAdmin"`
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"createdAt"`
}

func toUserView(u domain.User) userView {
	return userView{
		ID:        u.ID,
		Username:  u.Username,
		IsAdmin:   u.IsAdmin,
		Disabled:  u.Disabled(),
		CreatedAt: u.CreatedAt,
	}
}
