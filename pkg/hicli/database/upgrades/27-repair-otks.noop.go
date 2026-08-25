// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !go1.27 && !goexperiment.jsonv2

package upgrades

import (
	"context"

	"go.mau.fi/util/dbutil"
)

var upgradeV27 = dbutil.WrapUpgrade(-1, 27, 10, "Don't mark OTKs as needing repair", dbutil.TxnModeOn, func(ctx context.Context, db *dbutil.Database) error {
	return nil
})
