//go:build integration

package mysql

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestRefreshTokenRotationIsAtomicAndSingleUse(t *testing.T) {
	dsn := os.Getenv("IDENTITY_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("IDENTITY_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	ctx := context.Background()
	user := &identity.User{Username: "integration-user", Email: "integration@example.com", PasswordHash: "hash"}
	db.WithContext(ctx).Where("email = ?", user.Email).Delete(&userModel{})
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.WithContext(ctx).Delete(&userModel{}, user.ID) })
	old := identity.RefreshToken{UserID: user.ID, TokenHash: "old-token-hash", ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.CreateRefreshToken(ctx, old); err != nil {
		t.Fatal(err)
	}
	replacement := identity.RefreshToken{UserID: user.ID, TokenHash: "new-token-hash", ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.RotateRefreshToken(ctx, old.TokenHash, replacement); err != nil {
		t.Fatal(err)
	}
	if err := repo.RotateRefreshToken(ctx, old.TokenHash, replacement); err != identity.ErrNotFound {
		t.Fatalf("replay error = %v, want ErrNotFound", err)
	}
}
