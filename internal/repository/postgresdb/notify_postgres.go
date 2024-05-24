package postgresdb

import (
	"NotifiService/internal/entity"
	"NotifiService/internal/repository/repository_errors"
	"NotifiService/pkg/postgres"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
)

const (
	usersTable  = "users"
	notifyTable = "notifications"
)

type NotifyRepo struct {
	db *postgres.Postgres
}

func NewNotifyRepo(pg *postgres.Postgres) *NotifyRepo {
	return &NotifyRepo{pg}
}

func (r *NotifyRepo) GetUserCommunication(ctx context.Context, id string) (entity.UserCommunication, error) {
	sql, args, _ := r.db.Builder.
		Select("id", "email", "phone").
		From(usersTable).
		Where("id = ?", id).
		ToSql()

	var userCommunication entity.UserCommunication

	err := r.db.Pool.QueryRow(ctx, sql, args...).Scan(
		&userCommunication.ID,
		&userCommunication.Email,
		&userCommunication.Phone,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.UserCommunication{}, repository_errors.ErrNotFound
		}
		return entity.UserCommunication{}, fmt.Errorf("NotifyRepo.GetUserCommunication - r.Pool.QueryRow: %v", err)
	}

	return userCommunication, nil
}

func (r *NotifyRepo) EditUserPreferences(ctx context.Context, preferences entity.UserPreferences) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("NewNotifyRepo.EditUserPreferences - r.Pool.Begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sql, args, _ := r.db.Builder.
		Select("email_notify", "phone_notify").
		From(notifyTable).
		Where("user_id = ?", preferences.UserID).
		ToSql()

	var email_notify, phone_notify bool
	err = tx.QueryRow(ctx, sql, args...).Scan(&email_notify, &phone_notify)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository_errors.ErrNotFound
		}
		return fmt.Errorf("NewNotifyRepo.EditUserPreferences - tx.QueryRow: %v", err)
	}

	for i := 0; i < len(preferences.Preferences); i++ {
		if preferences.Preferences[i].Channel == "email" {
			email_notify = preferences.Preferences[i].Approval
		}
		if preferences.Preferences[i].Channel == "phone" {
			phone_notify = preferences.Preferences[i].Approval
		}
	}

	sql, args, _ = r.db.Builder.
		Update(notifyTable).
		Set("email_notify", email_notify).
		Set("phone_notify", phone_notify).
		Where("user_id = ?", preferences.UserID).
		ToSql()

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("AccountRepo.Deposit - tx.Exec: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("NewNotifyRepo.EditUserPreferences - tx.Commit: %v", err)
	}

	return nil
}
