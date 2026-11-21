package upgrades

import (
	"context"
	_ "embed"

	"go.mau.fi/util/dbutil"
)

//go:embed 34-thread-aggregates.sql.txt
var threadAggregatesSQL string

//go:embed 33-fix-own-reaction-redaction.sql
var fixOwnReactionRedactionSQL string

var upgradeV34 = dbutil.WrapUpgrade(-1, 34, 10, "Add thread aggregates", dbutil.TxnModeOn, func(ctx context.Context, db *dbutil.Database) error {
	hasThreadAggregates, err := db.ColumnExists(ctx, "event", "thread_count")
	if err != nil {
		return err
	} else if !hasThreadAggregates {
		_, err = db.Exec(ctx, threadAggregatesSQL)
		return err
	}
	hasUnknownTopLevel, err := db.ColumnExists(ctx, "event", "unknown_top_level")
	if err != nil {
		return err
	} else if !hasUnknownTopLevel {
		// Thread aggregates were applied as v32, upstream v32 was skipped (v33 ran normally)
		_, err = db.Exec(ctx, "ALTER TABLE event ADD COLUMN unknown_top_level TEXT")
	} else {
		// Thread aggregates were applied as v33, upstream v33 was skipped
		_, err = db.Exec(ctx, fixOwnReactionRedactionSQL)
	}
	return err
})
