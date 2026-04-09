package postgres

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ViewRefresher struct {
	db       *pgxpool.Pool
	interval time.Duration
}

func NewViewRefresher(db *pgxpool.Pool, interval time.Duration) *ViewRefresher {
	return &ViewRefresher{db: db, interval: interval}
}

func (r *ViewRefresher) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.refresh(ctx)
			}
		}
	}()
	log.Printf("view refresher started - interval: %s", r.interval)
}

func (r *ViewRefresher) refresh(ctx context.Context) {
	start := time.Now()
	_, err := r.db.Exec(ctx, `SELECT refresh_analytics_views()`)
	if err != nil {
		log.Printf("view refresh failed: %v", err)
		return
	}
	log.Printf("analytics views refreshed in %s", time.Since(start))
}
