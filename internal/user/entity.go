package user

import "time"

type Role string

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID              int
	Username        string
	Email           string
	Password        string
	Role            Role
	EmailVerifiedAt *time.Time
	PendingEmail    string
	TokenVersion    int
}

func (u User) IsAdmin() bool       { return u.Role == RoleAdmin }
func (u User) EmailVerified() bool { return u.EmailVerifiedAt != nil }

func ParseRole(value string) (Role, error) {
	switch Role(value) {
	case RoleUser:
		return RoleUser, nil
	case RoleAdmin:
		return RoleAdmin, nil
	default:
		return "", ErrInvalidRole
	}
}

func (u User) CanPromote() bool { return u.EmailVerified() }

func (u *User) UpdateProfile(username, email string) error {
	candidate := &User{Username: username, Email: email}
	if err := NormalizeAndValidate(candidate, false); err != nil {
		return err
	}

	u.Username = candidate.Username
	if candidate.Email == u.Email {
		u.PendingEmail = ""
	} else {
		u.PendingEmail = candidate.Email
	}

	return nil
}

func (u *User) AssignRole(role Role) error {
	if role == RoleAdmin && !u.CanPromote() {
		return ErrForbidden
	}

	u.Role = role

	return nil
}

func (u *User) VerifyEmail(at time.Time) bool {
	first := !u.EmailVerified()
	if u.PendingEmail != "" {
		u.Email = u.PendingEmail
		u.PendingEmail = ""
	}

	u.EmailVerifiedAt = &at
	u.TokenVersion++

	return first
}

func (u *User) ChangePasswordHash(hash string) {
	u.Password = hash
	u.TokenVersion++
}
