package cleanup

import (
	"context"
)

// Execute runs cleanup of old outbox events
func (c *Cleaner) Execute(ctx context.Context) error {
	c.logger.Info("Cleanup started")
	rowsDeleted, err := c.db.Outbox().Cleanup(ctx, c.retentionDays)
	if err != nil {
		c.logger.WithError(err).Error("cleanup failed")
		return err
	}

	c.db.Metrics().IncrementMetric(ctx, "outbox_cleaned", rowsDeleted)

	c.logger.WithField("rows_deleted", rowsDeleted).Info("outbox cleanup completed")
	return nil
}
