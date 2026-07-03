package mysqladapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestUserSecurityTransactions(t *testing.T) {
	dsn := os.Getenv("USER_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("USER_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewUserRepository(db)
	ctx := context.Background()
	suffix := uuid.NewString()
	t.Run("registration rolls back when verification dispatch fails", func(t *testing.T) {
		account := &user.User{
			Username: "rollback_" + suffix[:8], Email: fmt.Sprintf("rollback-%s@example.com", suffix),
			Password: "hashed", Role: user.RoleUser, TokenVersion: 1,
		}
		dispatchErr := errors.New("queue unavailable")
		err := repo.RegisterUser(ctx, account, &auth.EmailVerificationToken{
			TokenHash: "rollback-token-" + suffix, ExpiresAt: time.Now().Add(time.Hour),
		}, func() error { return dispatchErr })
		if !errors.Is(err, dispatchErr) {
			t.Fatalf("registration error = %v", err)
		}
		if _, err := repo.GetUserByEmail(ctx, account.Email); !errors.Is(err, user.ErrNotFound) {
			t.Fatalf("rolled-back user lookup error = %v", err)
		}
	})
	adminEmail := fmt.Sprintf("admin-%s@example.com", suffix)
	admin := &user.User{Username: "admin_" + suffix[:8], Email: adminEmail, Password: "hashed", Role: user.RoleUser, TokenVersion: 1}
	member := &user.User{Username: "member_" + suffix[:8], Email: fmt.Sprintf("member-%s@example.com", suffix), Password: "hashed", Role: user.RoleUser, TokenVersion: 1}
	for _, account := range []*user.User{admin, member} {
		if err := repo.Create(ctx, account); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Where("id = ?", account.ID).Delete(&userModel{}).Error })
	}

	for _, test := range []struct {
		account   *user.User
		hash      string
		bootstrap string
	}{
		{admin, "admin-token-hash-" + suffix, adminEmail},
		{member, "member-token-hash-" + suffix, adminEmail},
	} {
		if err := repo.ReplaceEmailVerificationToken(ctx, &auth.EmailVerificationToken{
			UserID: test.account.ID, TokenHash: test.hash, ExpiresAt: time.Now().Add(time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
		result, err := repo.VerifyEmail(ctx, test.hash, time.Now(), func(account user.User, admins int64) (user.Role, error) {
			if account.Email == test.bootstrap && admins == 0 {
				return user.RoleAdmin, nil
			}
			return account.Role, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.FirstVerification {
			t.Fatal("initial verification was not marked as first verification")
		}
	}
	verifiedAdmin, err := repo.GetByID(ctx, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if verifiedAdmin.Role != user.RoleAdmin || !verifiedAdmin.EmailVerified() {
		t.Fatalf("admin = %+v", verifiedAdmin)
	}
	if _, err := repo.VerifyEmail(ctx, "admin-token-hash-"+suffix, time.Now(), nil); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("replayed verification error = %v", err)
	}
	pendingEmail := fmt.Sprintf("updated-%s@example.com", suffix)
	if err := repo.UpdateProfile(ctx, member.ID, member.Username, pendingEmail); err != nil {
		t.Fatal(err)
	}
	pendingHash := "pending-token-hash-" + suffix
	if err := repo.ReplaceEmailVerificationToken(ctx, &auth.EmailVerificationToken{
		UserID: member.ID, TokenHash: pendingHash, ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	pendingResult, err := repo.VerifyEmail(ctx, pendingHash, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if pendingResult.FirstVerification {
		t.Fatal("pending email verification was marked as first verification")
	}

	if err := repo.ChangeRole(ctx, admin.ID, member.ID, user.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := repo.ChangeRole(ctx, member.ID, admin.ID, user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := repo.ChangeRole(ctx, admin.ID, member.ID, user.RoleUser); !errors.Is(err, user.ErrForbidden) {
		t.Fatalf("non-admin actor error = %v", err)
	}
	if err := repo.ChangeRole(ctx, member.ID, member.ID, user.RoleUser); !errors.Is(err, user.ErrLastAdmin) {
		t.Fatalf("last admin error = %v", err)
	}
}
