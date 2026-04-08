// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package database

import (
	"context"
	"encoding/json"
	"time"

	"go.mau.fi/util/dbutil"
	"go.mau.fi/util/jsontime"
)

const (
	getNonExpiredPushTargets = `
		SELECT device_id, type, data, encryption, expiration
		FROM push_registration
		WHERE expiration > $1
	`
	putPushRegistration = `
		INSERT INTO push_registration (device_id, type, data, encryption, expiration)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (device_id) DO UPDATE SET
			type = EXCLUDED.type,
			data = EXCLUDED.data,
			encryption = EXCLUDED.encryption,
			expiration = EXCLUDED.expiration
	`
)

type PushRegistrationQuery struct {
	*dbutil.QueryHelper[*PushRegistration]
}

func (prq *PushRegistrationQuery) Put(ctx context.Context, reg *PushRegistration) error {
	return prq.Exec(ctx, putPushRegistration, reg.sqlVariables()...)
}

func (seq *PushRegistrationQuery) GetAll(ctx context.Context) ([]*PushRegistration, error) {
	return seq.QueryMany(ctx, getNonExpiredPushTargets, time.Now().Unix())
}

type PushType string

const (
	PushTypeFCM  PushType = "fcm"
	PushTypeWeb  PushType = "web"
	PushTypeNtfy PushType = "ntfy"
)

type EncryptionKey struct {
	// 32 random bytes used as the static AES-GCM key.
	Key []byte `json:"key,omitempty"`
}

type PushRegistration struct {
	// An arbitrary (but stable) device identifier. Only one push registration can be active per device ID.
	DeviceID string `json:"device_id"`
	// The type of pusher.
	Type PushType `json:"type"`
	// The type-specific data.
	//
	// For FCM, this is the FCM token as a string.
	// For web push, this is the subscription info as a JSON object
	// (`endpoint` string and `keys` object with `p256dh` and `auth` strings).
	Data json.RawMessage `json:"data"`
	// An optional gomuks-specific encryption configuration. Mostly relevant for FCM (and APNs in
	// the future), as web push has built-in encryption.
	Encryption EncryptionKey `json:"encryption"`
	// Unix timestamp (seconds) when the registration should be considered stale.
	// The frontend should re-register well before this time.
	Expiration jsontime.Unix `json:"expiration"`
}

func (pe *PushRegistration) Scan(row dbutil.Scannable) (*PushRegistration, error) {
	err := row.Scan(&pe.DeviceID, &pe.Type, (*[]byte)(&pe.Data), dbutil.JSON{Data: &pe.Encryption}, &pe.Expiration)
	if err != nil {
		return nil, err
	}
	return pe, nil
}

func (pe *PushRegistration) sqlVariables() []any {
	if pe.Expiration.IsZero() {
		pe.Expiration = jsontime.U(time.Now().Add(7 * 24 * time.Hour))
	}
	return []interface{}{pe.DeviceID, pe.Type, unsafeJSONString(pe.Data), dbutil.JSON{Data: &pe.Encryption}, pe.Expiration}
}
