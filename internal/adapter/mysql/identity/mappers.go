package identity

import (
	"errors"
	"strings"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
	"gorm.io/gorm"
)

func fromUser(account *user.User) userModel {
	return userModel{
		ID: account.ID, Username: account.Username, Email: account.Email, Password: account.Password, Role: account.Role,
		EmailVerifiedAt: account.EmailVerifiedAt, PendingEmail: account.PendingEmail, TokenVersion: account.TokenVersion,
	}
}

func toUser(record userModel) *user.User {
	return &user.User{
		ID: record.ID, Username: record.Username, Email: record.Email, Password: record.Password, Role: record.Role,
		EmailVerifiedAt: record.EmailVerifiedAt, PendingEmail: record.PendingEmail, TokenVersion: record.TokenVersion,
	}
}

func mapUserNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return user.ErrNotFound
	}

	return err
}

func mapRecordNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}

func mapWriteError(err error) error {
	var mysqlErr *mysqldriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		if strings.Contains(mysqlErr.Message, "username") {
			return user.ErrDuplicateUsername
		}

		return user.ErrDuplicateEmail
	}

	return err
}

func requireAffected(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return user.ErrNotFound
	}

	return nil
}

// ErrNotFound indicates a refresh-token row was not found.
var ErrNotFound = errors.New("record not found")
