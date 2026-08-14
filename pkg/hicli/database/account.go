// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix/id"
)

const (
	getAccountQuery = `
		SELECT user_id, device_id, access_token, homeserver_url, next_batch,
		       client_id, refresh_token, expiry, displayname, avatar_url
		FROM account WHERE user_id = $1
	`
	putNextBatchQuery    = `UPDATE account SET next_batch = $2 WHERE user_id = $1`
	putRefreshTokenQuery = `UPDATE account SET refresh_token = $2, access_token = $3, expiry = $4 WHERE user_id = $1`
	putProfileQuery      = `UPDATE account SET displayname = $2, avatar_url = $3 WHERE user_id = $1`
	upsertAccountQuery   = `
		INSERT INTO account (
			user_id, device_id, access_token, homeserver_url, next_batch,
			client_id, refresh_token, expiry, displayname, avatar_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (user_id)
			DO UPDATE SET device_id = excluded.device_id,
			              access_token = excluded.access_token,
			              homeserver_url = excluded.homeserver_url,
			              next_batch = excluded.next_batch,
			              client_id = excluded.client_id,
			              refresh_token = excluded.refresh_token,
			              expiry = excluded.expiry,
			              displayname = excluded.displayname,
			              avatar_url = excluded.avatar_url
	`
)

type AccountQuery struct {
	*dbutil.QueryHelper[*Account]
}

func (aq *AccountQuery) GetFirstUserID(ctx context.Context) (userID id.UserID, err error) {
	var exists bool
	if exists, err = aq.GetDB().TableExists(ctx, "account"); err != nil || !exists {
		return
	}
	err = aq.GetDB().QueryRow(ctx, `SELECT user_id FROM account LIMIT 1`).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

func (aq *AccountQuery) Get(ctx context.Context, userID id.UserID) (*Account, error) {
	return aq.QueryOne(ctx, getAccountQuery, userID)
}

func (aq *AccountQuery) PutNextBatch(ctx context.Context, userID id.UserID, nextBatch string) error {
	return aq.Exec(ctx, putNextBatchQuery, userID, nextBatch)
}

func (aq *AccountQuery) PutRefreshToken(ctx context.Context, userID id.UserID, refreshToken, accessToken string, expiry time.Time) error {
	return aq.Exec(ctx, putRefreshTokenQuery, userID, refreshToken, accessToken, expiry.UnixMilli())
}

func (aq *AccountQuery) PutProfile(ctx context.Context, userID id.UserID, displayname string, avatarURL id.ContentURI) error {
	return aq.Exec(ctx, putProfileQuery, userID, displayname, &avatarURL)
}

func (aq *AccountQuery) Put(ctx context.Context, account *Account) error {
	return aq.Exec(ctx, upsertAccountQuery, account.sqlVariables()...)
}

type Account struct {
	UserID        id.UserID   `json:"user_id,omitempty"`
	DeviceID      id.DeviceID `json:"device_id,omitempty"`
	AccessToken   string      `json:"access_token,omitempty"`
	HomeserverURL string      `json:"homeserver_url,omitempty"`
	NextBatch     string      `json:"-"`

	ClientID     string             `json:"client_id,omitempty"`
	RefreshToken string             `json:"refresh_token,omitempty"`
	Expiry       jsontime.UnixMilli `json:"expiry,omitzero"`

	DisplayName string        `json:"display_name,omitempty"`
	AvatarURL   id.ContentURI `json:"avatar_url,omitempty"`
}

func (a *Account) Scan(row dbutil.Scannable) (*Account, error) {
	return dbutil.ValueOrErr(a, row.Scan(
		&a.UserID, &a.DeviceID, &a.AccessToken, &a.HomeserverURL, &a.NextBatch,
		&a.ClientID, &a.RefreshToken, &a.Expiry, &a.DisplayName, &a.AvatarURL,
	))
}

func (a *Account) sqlVariables() []any {
	return []any{
		a.UserID, a.DeviceID, a.AccessToken, a.HomeserverURL, a.NextBatch,
		a.ClientID, a.RefreshToken, a.Expiry, a.DisplayName, &a.AvatarURL,
	}
}
