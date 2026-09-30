// Package scheduler runs jobs periodically or at a daily local time.
//
// A Job pairs a Schedule with a function. Run blocks until the context is
// cancelled; Go starts it in a goroutine. The next run is always computed from
// the clock after the previous run finished, so a slow job skips missed slots
// instead of piling up. There is no jitter. Errors are logged and a panic in a
// job is recovered and logged, so a bad job never takes the process down.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// Clock abstracts time so tests can drive the scheduler.
type Clock interface {
	Now() time.Time
	// After returns a channel that receives once d has elapsed.
	After(d time.Duration) <-chan time.Time
}

// RealClock is the wall clock.
type RealClock struct{}

// Now implements Clock.
func (RealClock) Now() time.Time { return time.Now() }

// After implements Clock.
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Schedule computes run times.
type Schedule interface {
	// Next returns the first run time strictly after t.
	Next(t time.Time) time.Time
}

type every time.Duration

func (e every) Next(t time.Time) time.Time { return t.Add(time.Duration(e)) }

// Every runs the job every d (d must be positive).
func Every(d time.Duration) Schedule {
	if d <= 0 {
		panic("scheduler: Every needs a positive duration")
	}
	return every(d)
}

type dailyAt struct {
	hour, min int
	loc       *time.Location
}

func (d dailyAt) Next(t time.Time) time.Time {
	local := t.In(d.loc)
	y, m, day := local.Date()
	next := time.Date(y, m, day, d.hour, d.min, 0, 0, d.loc)
	for !next.After(t) {
		day++
		next = time.Date(y, m, day, d.hour, d.min, 0, 0, d.loc)
	}
	return next
}

// DailyAt runs the job every day at the wall-clock time hour:min in loc. In
// DST gaps the runtime normalises the time (02:30 becomes 03:30); in DST
// overlaps the first occurrence is used.
func DailyAt(hour, min int, loc *time.Location) Schedule {
	if hour < 0 || hour > 23 || min < 0 || min > 59 || loc == nil {
		panic("scheduler: invalid DailyAt arguments")
	}
	return dailyAt{hour: hour, min: min, loc: loc}
}

// Job is a named unit of work.
type Job struct {
	Name     string
	Schedule Schedule
	// RunOnStart runs the job once immediately before waiting for the schedule.
	RunOnStart bool
	Run        func(ctx context.Context) error
}

// Scheduler runs jobs. The zero Clock is the wall clock, the zero Log is
// slog.Default().
type Scheduler struct {
	Clock Clock
	Log   *slog.Logger
}

func (s *Scheduler) clock() Clock {
	if s.Clock == nil {
		return RealClock{}
	}
	return s.Clock
}

func (s *Scheduler) log() *slog.Logger {
	if s.Log == nil {
		return slog.Default()
	}
	return s.Log
}

// Run executes the job until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context, job Job) {
	clk := s.clock()
	if job.RunOnStart && ctx.Err() == nil {
		s.runOnce(ctx, job)
	}
	for {
		now := clk.Now()
		wait := job.Schedule.Next(now).Sub(now)
		select {
		case <-ctx.Done():
			return
		case <-clk.After(wait):
		}
		if ctx.Err() != nil {
			return
		}
		s.runOnce(ctx, job)
	}
}

// Go starts Run in a goroutine registered on wg (which may be nil).
func (s *Scheduler) Go(ctx context.Context, wg *sync.WaitGroup, job Job) {
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		s.Run(ctx, job)
	}()
}

func (s *Scheduler) runOnce(ctx context.Context, job Job) {
	log := s.log().With("job", job.Name)
	defer func() {
		if r := recover(); r != nil {
			log.Error("job panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
	}()
	if err := job.Run(ctx); err != nil && ctx.Err() == nil {
		log.Error("job failed", "err", err)
	}
}
