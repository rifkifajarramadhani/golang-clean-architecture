package user

import "time"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID              int
	Username        string
	Email           string
	Password        string
	Role            string
	EmailVerifiedAt *time.Time
	PendingEmail    string
	TokenVersion    int
}

func (u User) IsAdmin() bool       { return u.Role == RoleAdmin }
func (u User) EmailVerified() bool { return u.EmailVerifiedAt != nil }
