// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build go1.27 || goexperiment.jsonv2

package upgrades

import (
	"context"

	"go.mau.fi/util/dbutil"
)

var upgradeV27 = dbutil.WrapUpgrade(-1, 27, 10, "Mark OTKs as needing repair", dbutil.TxnModeOn, func(ctx context.Context, db *dbutil.Database) error {
	exists, err := db.TableExists(ctx, "crypto_account")
	if err != nil || !exists {
		return err
	}
	var shared bool
	err = db.QueryRow(ctx, "SELECT shared FROM crypto_account LIMIT 1").Scan(&shared)
	if err != nil || !shared {
		return err
	}
	var accountExists bool
	err = db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM account)").Scan(&accountExists)
	if err != nil || !accountExists {
		return err
	}
	_, err = db.Exec(ctx, `CREATE TABLE otks_need_reset (noop)`)
	return err
})
