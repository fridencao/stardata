package river

import (
	"context"

	"github.com/fridencao/stardata/admin"
	"github.com/riverqueue/river"
	"go.uber.org/zap"
)

// StarData Phase 5: reclaim editing locks whose holder stopped heartbeating.
//
// Reads already ignore expired rows, so this sweeper is housekeeping rather than
// correctness — it keeps the table from accumulating dead locks. Correctness comes
// from expires_at being checked at query time.
type DeleteExpiredEditingLocksArgs struct{}

func (DeleteExpiredEditingLocksArgs) Kind() string { return "delete_expired_editing_locks" }

type DeleteExpiredEditingLocksWorker struct {
	river.WorkerDefaults[DeleteExpiredEditingLocksArgs]
	admin *admin.Service
}

func (w *DeleteExpiredEditingLocksWorker) Work(ctx context.Context, job *river.Job[DeleteExpiredEditingLocksArgs]) error {
	n, err := w.admin.DB.DeleteExpiredEditingLocks(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		w.admin.Logger.Info("reclaimed expired editing locks", zap.Int("count", n))
	}
	return nil
}
