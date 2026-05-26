package cronjob

// scheduler.go — lightweight cron-job runner.
//
// A Scheduler runs registered jobs on their configured intervals using a
// single goroutine per job. It is intentionally minimal: no persistence, no
// distributed locking. For background metric collection this is sufficient.
//
// Usage:
//
//	s := cronjob.NewScheduler()
//	s.Register("metrics", 30*time.Second, myJob)
//	s.Start(ctx)
//	defer s.Stop()

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Job is a function executed by the Scheduler on every tick.
type Job func(ctx context.Context)

type entry struct {
	name     string
	interval time.Duration
	fn       Job
}

// Scheduler manages a set of periodic background jobs.
type Scheduler struct {
	jobs   []entry
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewScheduler creates a ready-to-use Scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{}
}

// Register adds a job that runs every interval.
// Must be called before Start().
func (s *Scheduler) Register(name string, interval time.Duration, fn Job) {
	s.jobs = append(s.jobs, entry{name: name, interval: interval, fn: fn})
}

// Start launches all registered jobs in background goroutines.
// ctx is used for the jobs themselves; Stop() cancels it.
func (s *Scheduler) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel

	for _, j := range s.jobs {
		s.wg.Add(1)
		go func(e entry) {
			defer s.wg.Done()
			s.runJob(ctx, e)
		}(j)
	}
}

// Stop signals all jobs to finish and waits for them to exit.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

func (s *Scheduler) runJob(ctx context.Context, e entry) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	slog.Info("cronjob started",
		slog.String("job", e.name),
		slog.Duration("interval", e.interval),
	)

	for {
		select {
		case <-ctx.Done():
			slog.Info("cronjob stopped",
				slog.String("job", e.name),
			)
			return
		case t := <-ticker.C:
			start := t
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("cronjob panic",
							slog.String("job", e.name),
							slog.Any("panic", r),
						)
					}
				}()
				e.fn(ctx)
			}()
			slog.Debug("cronjob tick",
				slog.String("job", e.name),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		}
	}
}
