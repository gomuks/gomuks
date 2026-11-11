
package upgrades

import (
	"context"
	_ "embed"

	"go.mau.fi/util/dbutil"
)

//go:embed 33-thread-aggregates.sql.txt
var threadAggregatesSQL string

var upgradeV33 = dbutil.WrapUpgrade(-1, 33, 10, "Add thread aggregates", dbutil.TxnModeOn, func(ctx context.Context, db *dbutil.Database) error {
	hasThreadAggregates, err := db.ColumnExists(ctx, "event", "thread_count")
	if err != nil {
		return err
	} else if hasThreadAggregates {
		_, err = db.Exec(ctx, "ALTER TABLE event ADD COLUMN unknown_top_level TEXT")
	} else {
		_, err = db.Exec(ctx, threadAggregatesSQL)
	}
	return err
})
