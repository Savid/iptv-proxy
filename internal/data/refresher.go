package data

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// Refresher manages periodic data refresh cycles in the background.
type Refresher struct {
	store    *Store
	fetcher  *Fetcher
	interval time.Duration
	logger   *logrus.Logger
}

// NewRefresher creates a new refresh manager.
func NewRefresher(store *Store, fetcher *Fetcher, interval time.Duration, logger *logrus.Logger) *Refresher {
	return &Refresher{
		store:    store,
		fetcher:  fetcher,
		interval: interval,
		logger:   logger,
	}
}

// Start begins the refresh cycle in a goroutine, stopping when the context is cancelled.
func (r *Refresher) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Refresh manager shutting down")
			return
		case <-ticker.C:
			err := r.refresh()
			nextInterval := r.scheduleNextRefresh(err)
			if nextInterval != r.interval {
				// Reset ticker with new interval for backoff
				ticker.Reset(nextInterval)
			}
		}
	}
}

func (r *Refresher) refresh() error {
	r.logger.Info("Starting data refresh")

	result, err := r.fetcher.FetchAll()
	if err != nil {
		r.logger.WithError(err).Error("Failed to refresh data")
		return err
	}

	// Update store only on successful fetch
	r.store.SetM3U(result.M3U.Raw, result.M3U.Channels)
	r.store.SetEPG(result.EPG.Raw, result.EPG.Filtered)

	r.logger.Info("Data refresh completed successfully")
	return nil
}

func (r *Refresher) scheduleNextRefresh(lastError error) time.Duration {
	if lastError == nil {
		// Success - use normal interval
		return r.interval
	}

	// Error - implement exponential backoff with max 5 minutes
	backoffDuration := r.interval / 2
	if backoffDuration > 5*time.Minute {
		backoffDuration = 5 * time.Minute
	}

	r.logger.WithField("interval", backoffDuration).Warn("Using backoff interval due to refresh error")
	return backoffDuration
}
