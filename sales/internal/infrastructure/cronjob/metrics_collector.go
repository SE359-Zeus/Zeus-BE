package cronjob

// metrics_collector.go — periodic job that snapshots all application metrics
// and logs them as structured slog records so Alloy/Loki/Grafana can graph them.
//
// The job also updates any derived gauges (e.g. active_sessions) that are
// cheaper to compute on a schedule than on every request.
//
// Register with the Scheduler in main():
//
//	scheduler.Register("metrics", 30*time.Second,
//	    cronjob.MetricsCollectorJob(obs.Metrics))

import (
	"context"

	"zeus-sales-service/internal/infrastructure/observability"
)

// MetricsCollectorJob returns a Job that snapshots the registry and logs it.
// interval is controlled by the Scheduler registration, not this function.
func MetricsCollectorJob(reg *observability.Registry) Job {
	return func(ctx context.Context) {
		snap := reg.Snapshot()
		observability.LogSnapshot(snap)
	}
}
