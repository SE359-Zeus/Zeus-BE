package observability

// log.go — slog.Handler that ships log entries to Grafana Alloy via the
// Loki HTTP push protocol (/loki/api/v1/push).
//
// The entry point is Alloy, NOT Loki directly.  Alloy collects the push,
// enriches/relabels the stream, and forwards it to Grafana Cloud Loki.
//
// Design decisions:
//   - Uses only the Go standard library (net/http, encoding/json, sync).
//   - Async batching: entries are queued in a buffered channel and flushed
//     when the batch is full or the flush interval elapses.
//   - Delivery failures are written to stderr; the primary stdout handler is
//     always the authoritative log sink.
//   - Labels are intentionally low-cardinality (job, env). High-cardinality
//     values (user_id, trace_id, …) live in the log line body, not labels.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ── Wire types that match the Alloy / Loki push API ──────────────────────────

// logEntry is a single [timestamp, line] pair.
type logEntry [2]string // [0] nanosecond unix ts string, [1] JSON log line

type logStream struct {
	Stream map[string]string `json:"stream"`
	Values []logEntry        `json:"values"`
}

type logPushBody struct {
	Streams []logStream `json:"streams"`
}

// ── Handler configuration ─────────────────────────────────────────────────────

// LogHandlerOptions configures the LogHandler.
type LogHandlerOptions struct {
	// PushURL is the Alloy OTLP/Loki receiver endpoint,
	// e.g. "http://alloy:3100/loki/api/v1/push"
	// or   "https://logs-prod-XXX.grafana.net/loki/api/v1/push" (direct cloud)
	PushURL string

	// BasicAuthUser / BasicAuthPass are used for Grafana Cloud authentication.
	// Leave both empty for a local / unauthenticated Alloy instance.
	BasicAuthUser string
	BasicAuthPass string

	// Labels are low-cardinality stream labels attached to every entry.
	// Recommended minimum: {"job": "zeus-scm", "env": "production"}
	Labels map[string]string

	// Level is the minimum slog level forwarded to Alloy (default: Info).
	Level slog.Level

	// BatchSize is the max entries before an automatic flush (default: 200).
	BatchSize int

	// FlushInterval is the max time between flushes (default: 5s).
	FlushInterval time.Duration
}

// ── Handler implementation ────────────────────────────────────────────────────

// LogHandler implements slog.Handler and ships log records to Alloy.
type LogHandler struct {
	opts   LogHandlerOptions
	client *http.Client
	queue  chan logEntry
	done   chan struct{}
	once   sync.Once
	attrs  []slog.Attr
	groups []string
}

// NewLogHandler creates the handler and starts its background flush goroutine.
// The returned stop function must be called on shutdown to flush pending entries.
func NewLogHandler(opts LogHandlerOptions) (h *LogHandler, stop func()) {
	if opts.BatchSize <= 0 {
		opts.BatchSize = 200
	}
	if opts.FlushInterval <= 0 {
		opts.FlushInterval = 5 * time.Second
	}

	h = &LogHandler{
		opts:   opts,
		client: &http.Client{Timeout: 10 * time.Second},
		queue:  make(chan logEntry, opts.BatchSize*2),
		done:   make(chan struct{}),
	}

	go h.runFlusher()

	return h, func() {
		h.once.Do(func() { close(h.done) })
	}
}

// Enabled satisfies slog.Handler.
func (h *LogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level
}

// Handle satisfies slog.Handler. It serialises the record to JSON and enqueues
// it. Enqueueing is non-blocking; entries are dropped if the queue is full.
func (h *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	m := make(map[string]any, r.NumAttrs()+6)
	m["time"] = r.Time.UTC().Format(time.RFC3339Nano)
	m["level"] = r.Level.String()
	m["msg"] = r.Message

	// Attach trace context if present.
	if id := TraceIDFromContext(ctx); id != "" {
		m["trace_id"] = id
	}
	if id := SpanIDFromContext(ctx); id != "" {
		m["span_id"] = id
	}

	for _, a := range h.attrs {
		flattenAttr(m, h.groups, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		flattenAttr(m, h.groups, a)
		return true
	})

	line, err := json.Marshal(m)
	if err != nil {
		return nil // silently drop malformed records
	}

	entry := logEntry{fmt.Sprintf("%d", r.Time.UnixNano()), string(line)}

	select {
	case h.queue <- entry:
	default:
		// Queue is full — drop rather than blocking the caller.
	}
	return nil
}

// WithAttrs satisfies slog.Handler.
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := h.clone()
	clone.attrs = append(clone.attrs, attrs...)
	return clone
}

// WithGroup satisfies slog.Handler.
func (h *LogHandler) WithGroup(name string) slog.Handler {
	clone := h.clone()
	clone.groups = append(clone.groups, name)
	return clone
}

// ── Internal ──────────────────────────────────────────────────────────────────

func (h *LogHandler) clone() *LogHandler {
	c := *h
	c.attrs = append([]slog.Attr(nil), h.attrs...)
	c.groups = append([]string(nil), h.groups...)
	return &c
}

func (h *LogHandler) runFlusher() {
	ticker := time.NewTicker(h.opts.FlushInterval)
	defer ticker.Stop()

	batch := make([]logEntry, 0, h.opts.BatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		h.push(batch)
		batch = batch[:0]
	}

	for {
		select {
		case e := <-h.queue:
			batch = append(batch, e)
			if len(batch) >= h.opts.BatchSize {
				flush()
			}

		case <-ticker.C:
			flush()

		case <-h.done:
		drain:
			for {
				select {
				case e := <-h.queue:
					batch = append(batch, e)
				default:
					break drain
				}
			}
			flush()
			return
		}
	}
}

// push ships a batch to the Alloy receiver endpoint. Errors go to stderr so
// they never loop back through the slog pipeline.
func (h *LogHandler) push(entries []logEntry) {
	body := logPushBody{
		Streams: []logStream{{Stream: h.opts.Labels, Values: entries}},
	}

	data, err := json.Marshal(body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log-handler: marshal error: %v\n", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, h.opts.PushURL, bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "log-handler: build request error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if h.opts.BasicAuthUser != "" {
		req.SetBasicAuth(h.opts.BasicAuthUser, h.opts.BasicAuthPass)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log-handler: push error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		fmt.Fprintf(os.Stderr, "log-handler: unexpected response status %d\n", resp.StatusCode)
	}
}

// flattenAttr writes a slog.Attr into a flat map, flattening group nesting
// with dot-separated keys (e.g. group "http" + key "status" → "http.status").
func flattenAttr(m map[string]any, groups []string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}

	key := a.Key
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}

	if a.Value.Kind() == slog.KindGroup {
		prefix := key
		if a.Key == "" {
			prefix = strings.Join(groups, ".")
		}
		for _, ga := range a.Value.Group() {
			flattenAttr(m, []string{prefix}, ga)
		}
		return
	}

	m[key] = a.Value.Any()
}
