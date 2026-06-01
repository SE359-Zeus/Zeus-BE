package observability

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Counter is a monotonically increasing int64.
type Counter struct{ v atomic.Int64 }

func (c *Counter) Inc()         { c.v.Add(1) }
func (c *Counter) Add(n int64)  { c.v.Add(n) }
func (c *Counter) Value() int64 { return c.v.Load() }

// Gauge is a float-precision value (stored ×1000 to avoid float atomics).
type Gauge struct{ v atomic.Int64 }

func (g *Gauge) Set(f float64)  { g.v.Store(int64(f * 1000)) }
func (g *Gauge) Inc()           { g.v.Add(1000) }
func (g *Gauge) Dec()           { g.v.Add(-1000) }
func (g *Gauge) Value() float64 { return float64(g.v.Load()) / 1000 }

// MetricSnapshot is a point-in-time reading of all metrics.
type MetricSnapshot struct {
	Timestamp time.Time
	Counters  map[string]int64
	Gauges    map[string]float64
}

// Registry holds all application metrics.
type Registry struct {
	mu       sync.RWMutex
	counters map[string]*Counter
	gauges   map[string]*Gauge
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[string]*Counter),
		gauges:   make(map[string]*Gauge),
	}
}

// Counter returns (or creates) a named counter.
func (r *Registry) Counter(name string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.counters[name]; ok {
		return c
	}
	c := &Counter{}
	r.counters[name] = c
	return c
}

// Gauge returns (or creates) a named gauge.
func (r *Registry) Gauge(name string) *Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	if g, ok := r.gauges[name]; ok {
		return g
	}
	g := &Gauge{}
	r.gauges[name] = g
	return g
}

// Snapshot returns a point-in-time copy augmented with Go runtime stats.
func (r *Registry) Snapshot() MetricSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snap := MetricSnapshot{
		Timestamp: time.Now().UTC(),
		Counters:  make(map[string]int64, len(r.counters)),
		Gauges:    make(map[string]float64, len(r.gauges)+6),
	}
	for k, c := range r.counters {
		snap.Counters[k] = c.Value()
	}
	for k, g := range r.gauges {
		snap.Gauges[k] = g.Value()
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	snap.Gauges["go_goroutines"] = float64(runtime.NumGoroutine())
	snap.Gauges["go_heap_alloc_bytes"] = float64(ms.HeapAlloc)
	snap.Gauges["go_heap_sys_bytes"] = float64(ms.HeapSys)
	snap.Gauges["go_gc_pause_ns_total"] = float64(ms.PauseTotalNs)
	snap.Gauges["go_num_gc"] = float64(ms.NumGC)
	snap.Gauges["go_next_gc_bytes"] = float64(ms.NextGC)
	return snap
}

// WritePrometheusText writes Prometheus text-exposition format (0.0.4) to w.
func (r *Registry) WritePrometheusText(w io.Writer) {
	snap := r.Snapshot()

	cKeys := make([]string, 0, len(snap.Counters))
	for k := range snap.Counters {
		cKeys = append(cKeys, k)
	}
	sort.Strings(cKeys)

	gKeys := make([]string, 0, len(snap.Gauges))
	for k := range snap.Gauges {
		gKeys = append(gKeys, k)
	}
	sort.Strings(gKeys)

	for _, k := range cKeys {
		safe := prometheusName(k)
		fmt.Fprintf(w, "# TYPE %s counter\n%s %d\n", safe, safe, snap.Counters[k])
	}
	for _, k := range gKeys {
		safe := prometheusName(k)
		fmt.Fprintf(w, "# TYPE %s gauge\n%s %g\n", safe, safe, snap.Gauges[k])
	}
}

func prometheusName(s string) string {
	return strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(s)
}

// ── Well-known metric name constants ─────────────────────────────────────────

const (
	MetricHTTPRequestsTotal = "http_requests_total"
	MetricHTTPRequestErrors = "http_request_errors_total"
	MetricHTTPPanicsTotal   = "http_panics_total"

	MetricAuthLoginSuccess = "auth_login_success_total"
	MetricAuthLoginFailure = "auth_login_failure_total"
	MetricAuthRefreshTotal = "auth_refresh_total"
	MetricAuthLogoutTotal  = "auth_logout_total"

	MetricUserCreatedTotal = "user_created_total"

	MetricAuditIngestedTotal = "audit_ingested_total"

	MetricCacheHits   = "cache_hits_total"
	MetricCacheMisses = "cache_misses_total"
)

// DefaultRegistry is the service-wide singleton, initialised by Provider.
var DefaultRegistry = NewRegistry()

// MetricsHTTPHandler returns a net/http handler serving the Prometheus text format.
// Register at GET /metrics (internal / Alloy-only).
func MetricsHTTPHandler(reg *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		reg.WritePrometheusText(w)
	}
}

// LogSnapshot logs a metric snapshot as a structured slog Info record.
// Called by the metrics cronjob.
func LogSnapshot(snap MetricSnapshot) {
	args := []any{
		slog.String("event", "metrics_snapshot"),
		slog.String("ts", snap.Timestamp.Format(time.RFC3339)),
	}
	for k, v := range snap.Counters {
		args = append(args, slog.Int64(k, v))
	}
	for k, v := range snap.Gauges {
		args = append(args, slog.Float64(k, v))
	}
	slog.Info("metrics snapshot", args...)
}
