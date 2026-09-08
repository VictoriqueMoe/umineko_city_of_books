package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"umineko_city_of_books/internal/dao/sqlcgen"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
)

type (
	DeviceTokenDAO interface {
		Upsert(ctx context.Context, s spec.NewDeviceToken, tx ...*sql.Tx) error
		RegistrationsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.DeviceRegistration, error)
		Delete(ctx context.Context, s spec.DeviceTokenDeletion, tx ...*sql.Tx) error
		DeleteMany(ctx context.Context, s spec.DeviceTokenBulkDeletion, tx ...*sql.Tx) error
	}

	deviceTokenDAO struct {
		db *sql.DB
	}
)

func toDeviceRegistration(row sqlcgen.ListDeviceTokensForUserRow) model.DeviceRegistration {
	return model.DeviceRegistration{
		Token:    row.Token,
		Platform: row.Platform,
	}
}

func (r *deviceTokenDAO) Upsert(ctx context.Context, s spec.NewDeviceToken, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).UpsertDeviceToken(ctx, sqlcgen.UpsertDeviceTokenParams{
		Token:    s.Token,
		UserID:   s.UserID,
		Platform: s.Platform,
	})
	if err != nil {
		return fmt.Errorf("upsert device token: %w", err)
	}

	return nil
}

func (r *deviceTokenDAO) RegistrationsForUser(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) ([]model.DeviceRegistration, error) {
	rows, err := genQueries(r.db, tx).ListDeviceTokensForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list device tokens: %w", err)
	}

	var registrations []model.DeviceRegistration
	for _, row := range rows {
		registrations = append(registrations, toDeviceRegistration(row))
	}

	return registrations, nil
}

func (r *deviceTokenDAO) Delete(ctx context.Context, s spec.DeviceTokenDeletion, tx ...*sql.Tx) error {
	err := genQueries(r.db, tx).DeleteDeviceToken(ctx, sqlcgen.DeleteDeviceTokenParams{
		Token:  s.Token,
		UserID: s.UserID,
	})
	if err != nil {
		return fmt.Errorf("delete device token: %w", err)
	}

	return nil
}

func (r *deviceTokenDAO) DeleteMany(ctx context.Context, s spec.DeviceTokenBulkDeletion, tx ...*sql.Tx) error {
	for _, token := range s.Tokens {
		if err := r.Delete(ctx, spec.DeviceTokenDeletion{UserID: s.UserID, Token: token}, tx...); err != nil {
			return err
		}
	}

	return nil
}
