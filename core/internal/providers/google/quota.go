package google

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	"google.golang.org/api/googleapi"
)

// Google Calendar currently allows 600 requests/minute per user/project.
// Pace each account at 240/minute, with headroom for another desktop.
// Calendar and Tasks share this conservative local gate.
// https://developers.google.com/workspace/calendar/api/guides/quota
const quotaUnitsPerSecond = 4
const maxReadRetries = 5
const maxInlineDelay = 2 * time.Second
const maxInlineRetryWait = 5 * time.Second

// Providers are short-lived (sync, IPC and RSVP each build their own). Keep
// account budgets and server cooldowns for the lifetime of this process.
var accountQuotas sync.Map // account ID -> *quotaGate

func accountQuota(id string) *quotaGate {
	if id == "" {
		return newQuotaGate()
	}
	value, _ := accountQuotas.LoadOrStore(id, newQuotaGate())
	return value.(*quotaGate)
}

type deferredRetry struct {
	until time.Time
	now   func() time.Time
	cause error
}

func (e *deferredRetry) Error() string {
	return fmt.Sprintf("Google API retry deferred for %s", e.RetryAfter())
}
func (e *deferredRetry) Unwrap() error             { return e.cause }
func (e *deferredRetry) RetryAfter() time.Duration { return max(0, e.until.Sub(e.now())) }

type quotaGate struct {
	mu           sync.Mutex
	next         time.Time
	blockedUntil time.Time
	cause        error
	now          func() time.Time
	sleep        func(context.Context, time.Duration) error
}

func newQuotaGate() *quotaGate {
	return &quotaGate{now: time.Now, sleep: sleepContext}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Reserve only once a request can start. Waiters re-check shared cooldowns
// so one rate-limit response also slows concurrent search and user actions.
func (q *quotaGate) wait(ctx context.Context, cost int, spend func(time.Duration) bool) error {
	started := q.now()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		q.mu.Lock()
		now := q.now()
		ready := q.next
		if q.blockedUntil.After(ready) {
			ready = q.blockedUntil
		}
		delay := ready.Sub(now)
		cooldown := max(0, q.blockedUntil.Sub(now))
		if delay > 0 && (delay > maxInlineDelay || now.Sub(started)+delay > maxInlineDelay ||
			(cooldown > 0 && spend != nil && !spend(cooldown))) {
			err := &deferredRetry{until: ready, now: q.now, cause: q.cause}
			q.mu.Unlock()
			return err
		}
		if delay <= 0 {
			q.next = now.Add(time.Duration(cost) * time.Second / quotaUnitsPerSecond)
			q.mu.Unlock()
			return nil
		}
		q.mu.Unlock()
		if err := q.sleep(ctx, delay); err != nil {
			return err
		}
	}
}

func (q *quotaGate) cooldown(delay time.Duration, cause error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	until := q.now().Add(delay)
	if until.After(q.blockedUntil) {
		q.blockedUntil = until
		q.cause = cause
	}
}

func quotaLimited(err error) bool {
	var e *googleapi.Error
	if !errors.As(err, &e) {
		return false
	}
	if e.Code == http.StatusTooManyRequests {
		return true
	}
	if e.Code != http.StatusForbidden {
		return false
	}
	for _, detail := range e.Errors {
		if detail.Reason == "rateLimitExceeded" || detail.Reason == "userRateLimitExceeded" {
			return true
		}
	}
	return false
}

func readRetryDelay(err error, attempt int, now time.Time) (time.Duration, bool) {
	var e *googleapi.Error
	if !errors.As(err, &e) {
		return 0, false
	}
	limited := quotaLimited(err)
	if !limited && e.Code != 500 && e.Code != 502 && e.Code != 503 && e.Code != 504 {
		return 0, false
	}
	delay := time.Second * time.Duration(1<<min(attempt, 6))
	// Google's Retry-After is a lower bound, including HTTP-date values.
	if seconds, err := strconv.ParseInt(e.Header.Get("Retry-After"), 10, 32); err == nil && seconds > 0 {
		delay = max(delay, time.Duration(seconds)*time.Second)
	} else if when, err := http.ParseTime(e.Header.Get("Retry-After")); err == nil {
		delay = max(delay, when.Sub(now))
	}
	return delay, true
}

// Retry only reads, preserving pages already fetched in the current cycle.
// Mutations are paced but not replayed: inserts must not create duplicates.
func googleCall[T any](ctx context.Context, p *Provider, read bool, call func() (*T, error)) (*T, error) {
	q := p.quotaGate()
	for attempt := 0; ; attempt++ {
		if err := p.waitQuota(ctx); err != nil {
			return nil, err
		}
		result, err := call()
		if err == nil {
			return result, nil
		}
		delay, retry := readRetryDelay(err, attempt, q.now())
		if !retry {
			return nil, err
		}
		// Even a mutation or exhausted read must retain Google's cooldown.
		delay += time.Duration(rand.Int64N(int64(time.Second)))
		q.cooldown(delay, err)
		if !read || attempt >= maxReadRetries {
			return nil, &deferredRetry{until: q.now().Add(delay), now: q.now, cause: err}
		}
	}
}

func (p *Provider) quotaGate() *quotaGate {
	p.quotaOnce.Do(func() {
		if p.quota == nil {
			p.quota = accountQuota(p.account.ID)
		}
	})
	return p.quota
}

// One provider spans an entire sync cycle (all calendars and pages). Limit
// cumulative inline retry sleep for that cycle, not separately for each page.
func (p *Provider) waitQuota(ctx context.Context) error {
	return p.quotaGate().wait(ctx, 1, func(delay time.Duration) bool {
		p.retryMu.Lock()
		defer p.retryMu.Unlock()
		if delay > maxInlineRetryWait-p.retryWait {
			return false
		}
		p.retryWait += delay
		return true
	})
}
