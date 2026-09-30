package scheduler_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/scheduler"
)

// fakeClock fires timers only when Advance moves time past them.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []waiter
	changed chan struct{} // signalled whenever a waiter is added
}

type waiter struct {
	at time.Time
	ch chan time.Time
}

func newFakeClock(now time.Time) *fakeClock {
	return &fakeClock{now: now, changed: make(chan struct{}, 100)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	c.waiters = append(c.waiters, waiter{at: c.now.Add(d), ch: ch})
	c.changed <- struct{}{}
	return ch
}

// waitForWaiter blocks until the scheduler is waiting on the clock.
func (c *fakeClock) waitForWaiter(t *testing.T) {
	t.Helper()
	select {
	case <-c.changed:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler never waited on the clock")
	}
}

// Advance moves time forward and fires due timers.
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	rest := c.waiters[:0]
	for _, w := range c.waiters {
		if !w.at.After(c.now) {
			w.ch <- c.now
		} else {
			rest = append(rest, w)
		}
	}
	c.waiters = rest
}

func TestEvery(t *testing.T) {
	clk := newFakeClock(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	runs := make(chan time.Time, 10)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	s := &scheduler.Scheduler{Clock: clk, Log: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))}
	s.Go(ctx, &wg, scheduler.Job{
		Name:     "tick",
		Schedule: scheduler.Every(time.Hour),
		Run:      func(context.Context) error { runs <- clk.Now(); return nil },
	})

	clk.waitForWaiter(t)
	clk.Advance(59 * time.Minute)
	select {
	case <-runs:
		t.Fatal("ran too early")
	case <-time.After(20 * time.Millisecond):
	}
	clk.Advance(time.Minute)
	got := <-runs
	if want := time.Date(2026, 10, 1, 13, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("first run at %v, want %v", got, want)
	}
	clk.waitForWaiter(t)
	clk.Advance(time.Hour)
	<-runs

	cancel()
	wg.Wait() // Run must return on cancel
}

func TestRunOnStart(t *testing.T) {
	clk := newFakeClock(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	runs := make(chan struct{}, 10)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	s := &scheduler.Scheduler{Clock: clk}
	s.Go(ctx, &wg, scheduler.Job{
		Name:       "start",
		Schedule:   scheduler.Every(time.Hour),
		RunOnStart: true,
		Run:        func(context.Context) error { runs <- struct{}{}; return nil },
	})
	<-runs
	clk.waitForWaiter(t)
	cancel()
	wg.Wait()
}

func TestErrorsAndPanicsAreLogged(t *testing.T) {
	clk := newFakeClock(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	var mu sync.Mutex
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(lockedWriter{&mu, &buf}, nil))
	calls := make(chan int, 10)
	n := 0
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	s := &scheduler.Scheduler{Clock: clk, Log: log}
	s.Go(ctx, &wg, scheduler.Job{
		Name:     "flaky",
		Schedule: scheduler.Every(time.Minute),
		Run: func(context.Context) error {
			n++
			calls <- n
			if n == 1 {
				return errors.New("boom")
			}
			panic("kaboom")
		},
	})
	for i := 1; i <= 2; i++ {
		clk.waitForWaiter(t)
		clk.Advance(time.Minute)
		<-calls
	}
	clk.waitForWaiter(t) // scheduler survived both, waits again
	cancel()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	out := buf.String()
	if !strings.Contains(out, "job failed") || !strings.Contains(out, "boom") {
		t.Errorf("error not logged: %s", out)
	}
	if !strings.Contains(out, "job panicked") || !strings.Contains(out, "kaboom") {
		t.Errorf("panic not logged: %s", out)
	}
}

type lockedWriter struct {
	mu *sync.Mutex
	w  *bytes.Buffer
}

func (l lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func TestDailyAtNext(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	sched := scheduler.DailyAt(20, 30, berlin)
	tests := []struct {
		name string
		from time.Time
		want time.Time
	}{
		{"before", time.Date(2026, 10, 1, 10, 0, 0, 0, berlin), time.Date(2026, 10, 1, 20, 30, 0, 0, berlin)},
		{"exactly at", time.Date(2026, 10, 1, 20, 30, 0, 0, berlin), time.Date(2026, 10, 2, 20, 30, 0, 0, berlin)},
		{"after", time.Date(2026, 10, 1, 23, 0, 0, 0, berlin), time.Date(2026, 10, 2, 20, 30, 0, 0, berlin)},
		{"across DST end", time.Date(2026, 10, 24, 21, 0, 0, 0, berlin), time.Date(2026, 10, 25, 20, 30, 0, 0, berlin)},
		{"utc input", time.Date(2026, 10, 1, 19, 0, 0, 0, time.UTC), time.Date(2026, 10, 2, 20, 30, 0, 0, berlin)},
	}
	for _, tt := range tests {
		if got := sched.Next(tt.from); !got.Equal(tt.want) {
			t.Errorf("%s: Next = %v, want %v", tt.name, got, tt.want)
		}
	}
	// The wait across the DST end is 25 hours.
	from := time.Date(2026, 10, 24, 20, 30, 0, 0, berlin)
	if d := sched.Next(from).Sub(from); d != 25*time.Hour {
		t.Errorf("wait across DST end = %v, want 25h", d)
	}
}

func TestDailyAtRuns(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	clk := newFakeClock(time.Date(2026, 10, 1, 20, 0, 0, 0, berlin))
	runs := make(chan time.Time, 10)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	s := &scheduler.Scheduler{Clock: clk}
	s.Go(ctx, &wg, scheduler.Job{
		Name:     "daily",
		Schedule: scheduler.DailyAt(20, 30, berlin),
		Run:      func(context.Context) error { runs <- clk.Now(); return nil },
	})
	clk.waitForWaiter(t)
	clk.Advance(30 * time.Minute)
	if got := <-runs; !got.Equal(time.Date(2026, 10, 1, 20, 30, 0, 0, berlin)) {
		t.Errorf("ran at %v", got)
	}
	cancel()
	wg.Wait()
}
