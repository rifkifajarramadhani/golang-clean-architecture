package identity

import (
	"context"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) Create(ctx context.Context, account *user.User) error {
	return r.createUser(ctx, account)
}

func (r *Repository) createUser(ctx context.Context, account *user.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pending int64
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&userModel{}).
			Where("pending_email = ?", account.Email).Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return user.ErrDuplicateEmail
		}
		model := fromUser(account)
		if err := tx.Create(&model).Error; err != nil {
			return mapWriteError(err)
		}
		account.ID = model.ID
		return nil
	})
}

func (r *Repository) List(ctx context.Context, page, limit int) ([]*user.User, int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&userModel{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	var records []userModel
	if err := r.db.WithContext(ctx).Order("id ASC").Offset((page - 1) * limit).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	accounts := make([]*user.User, 0, len(records))
	for _, record := range records {
		accounts = append(accounts, toUser(record))
	}
	return accounts, count, nil
}

func (r *Repository) GetByID(ctx context.Context, id int) (*user.User, error) {
	return r.GetUserByID(ctx, id)
}

func (r *Repository) GetUserByID(ctx context.Context, id int) (*user.User, error) {
	var record userModel
	if err := r.db.WithContext(ctx).First(&record, id).Error; err != nil {
		return nil, mapUserNotFound(err)
	}
	return toUser(record), nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	var record userModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&record).Error; err != nil {
		return nil, mapUserNotFound(err)
	}
	return toUser(record), nil
}

func (r *Repository) GetUserByEmailOrPending(ctx context.Context, email string) (*user.User, error) {
	var record userModel
	if err := r.db.WithContext(ctx).Where("email = ? OR pending_email = ?", email, email).First(&record).Error; err != nil {
		return nil, mapUserNotFound(err)
	}
	return toUser(record), nil
}

func (r *Repository) CountAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&userModel{}).Where("role = ?", user.RoleAdmin).Count(&count).Error
	return count, err
}

func (r *Repository) UpdateProfile(ctx context.Context, id int, username, email string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record userModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&record, id).Error; err != nil {
			return mapUserNotFound(err)
		}
		updates := map[string]any{"username": username}
		if email == record.Email {
			updates["pending_email"] = ""
		} else {
			var count int64
			if err := tx.Model(&userModel{}).Where("(email = ? OR pending_email = ?) AND id <> ?", email, email, id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return user.ErrDuplicateEmail
			}
			updates["pending_email"] = email
		}
		return mapWriteError(tx.Model(&record).Updates(updates).Error)
	})
}

func (r *Repository) ChangePassword(ctx context.Context, id int, hashedPassword string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&userModel{}).Where("id = ?", id).Updates(map[string]any{
			"password": hashedPassword, "token_version": gorm.Expr("token_version + 1"),
		})
		if err := requireAffected(result); err != nil {
			return err
		}
		return tx.Model(&refreshTokenModel{}).Where("user_id = ? AND revoked_at IS NULL", id).Update("revoked_at", time.Now()).Error
	})
}

// ChangeRole re-checks the admin-actor, promote-requires-verified, and
// last-admin invariants that user.Service already validated: the service
// checks are advisory (fast failure, business rules in core), while these run
// under FOR UPDATE locks and are authoritative against concurrent changes.
func (r *Repository) ChangeRole(ctx context.Context, actorID, targetID int, role user.Role) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var actor, target userModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&actor, actorID).Error; err != nil {
			return mapUserNotFound(err)
		}
		if actor.Role != user.RoleAdmin {
			return user.ErrForbidden
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&target, targetID).Error; err != nil {
			return mapUserNotFound(err)
		}
		if role == user.RoleAdmin && target.EmailVerifiedAt == nil {
			return user.ErrForbidden
		}
		if target.Role == user.RoleAdmin && role != user.RoleAdmin {
			var admins []userModel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("role = ?", user.RoleAdmin).Find(&admins).Error; err != nil {
				return err
			}
			if len(admins) <= 1 {
				return user.ErrLastAdmin
			}
		}
		if err := tx.Model(&target).Updates(map[string]any{
			"role": role, "token_version": gorm.Expr("token_version + 1"),
		}).Error; err != nil {
			return err
		}
		return tx.Model(&refreshTokenModel{}).Where("user_id = ? AND revoked_at IS NULL", targetID).
			Update("revoked_at", time.Now()).Error
	})
}

// Delete re-checks the last-admin invariant under FOR UPDATE locks; see
// ChangeRole for why the user.Service pre-check is repeated here.
func (r *Repository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target userModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&target, id).Error; err != nil {
			return mapUserNotFound(err)
		}
		if target.Role == user.RoleAdmin {
			var admins []userModel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("role = ?", user.RoleAdmin).Find(&admins).Error; err != nil {
				return err
			}
			if len(admins) <= 1 {
				return user.ErrLastAdmin
			}
		}
		return requireAffected(tx.Delete(&target))
	})
}
