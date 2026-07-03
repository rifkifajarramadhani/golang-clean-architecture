package user

import "time"

// Role is the set of privileges assigned to a user.
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

// ParseRole validates and normalizes an externally supplied role.
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

// CanPromote reports whether the user satisfies the domain requirements for
// receiving administrator privileges.
func (u User) CanPromote() bool { return u.EmailVerified() }

// UpdateProfile applies a validated profile transition.
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

// AssignRole applies the domain checks for a role transition.
func (u *User) AssignRole(role Role) error {
	if role == RoleAdmin && !u.CanPromote() {
		return ErrForbidden
	}
	u.Role = role
	return nil
}

// VerifyEmail applies a successful email-verification transition.
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

// ChangePasswordHash records a password credential transition.
func (u *User) ChangePasswordHash(hash string) {
	u.Password = hash
	u.TokenVersion++
}
