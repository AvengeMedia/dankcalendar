package google

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	gtasks "google.golang.org/api/tasks/v1"
)

func fakeQuota() (*quotaGate, *[]time.Duration) {
	now := time.Unix(0, 0)
	var sleeps []time.Duration
	q := &quotaGate{now: func() time.Time { return now }}
	q.sleep = func(ctx context.Context, d time.Duration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		sleeps = append(sleeps, d)
		now = now.Add(d)
		return nil
	}
	return q, &sleeps
}

func TestQuotaPacesByCostAndHonorsCooldown(t *testing.T) {
	q, sleeps := fakeQuota()
	ctx := context.Background()
	for _, cost := range []int{4, 2, 1} {
		require.NoError(t, q.wait(ctx, cost, nil))
	}
	require.Equal(t, []time.Duration{time.Second, time.Second / 2}, *sleeps)
	q.cooldown(time.Minute, nil)
	q.cooldown(time.Second, nil)
	var deferred *deferredRetry
	require.ErrorAs(t, q.wait(ctx, 1, nil), &deferred)
	require.Equal(t, time.Minute, deferred.RetryAfter())
	require.Len(t, *sleeps, 2, "long cooldown slept inline")
}

func TestReadRetryDelay(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	cases := []struct {
		name  string
		err   error
		want  time.Duration
		retry bool
	}{
		{"quota", &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "rateLimitExceeded"}}}, time.Second, true},
		{"user quota", &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "userRateLimitExceeded"}}}, time.Second, true},
		{"429", &googleapi.Error{Code: 429}, time.Second, true},
		{"server", &googleapi.Error{Code: 503}, time.Second, true},
		{"retry seconds", &googleapi.Error{Code: 429, Header: http.Header{"Retry-After": []string{"180"}}}, 3 * time.Minute, true},
		{"retry date", &googleapi.Error{Code: 429, Header: http.Header{"Retry-After": []string{now.Add(4 * time.Minute).Format(http.TimeFormat)}}}, 4 * time.Minute, true},
		{"permission", &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "insufficientPermissions"}}}, 0, false},
		{"auth", &googleapi.Error{Code: 401}, 0, false},
		{"not found", &googleapi.Error{Code: 404}, 0, false},
		{"network", errors.New("offline"), 0, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			d, retry := readRetryDelay(tt.err, 0, now)
			require.Equal(t, tt.want, d)
			require.Equal(t, tt.retry, retry)
		})
	}
	d, _ := readRetryDelay(&googleapi.Error{Code: 429}, 4, now)
	require.Equal(t, 16*time.Second, d)
}

func TestReadRetriesAreBoundedAndCancellable(t *testing.T) {
	q, _ := fakeQuota()
	r := &Provider{quota: q}
	calls := 0
	_, err := googleCall(context.Background(), r, true, func() (*int, error) { calls++; return nil, &googleapi.Error{Code: 429} })
	require.Error(t, err)
	require.GreaterOrEqual(t, calls, 2)
	require.LessOrEqual(t, calls, 3)
	ctx, cancel := context.WithCancel(context.Background())
	calls = 0
	q, _ = fakeQuota()
	r = &Provider{quota: q}
	_, err = googleCall(ctx, r, true, func() (*int, error) { calls++; cancel(); return nil, &googleapi.Error{Code: 429} })
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
}

func TestSyncPagesRetriesOnlyFailedPage(t *testing.T) {
	for _, tasks := range []bool{false, true} {
		t.Run(map[bool]string{false: "calendar", true: "tasks"}[tasks], func(t *testing.T) {
			var pages []string
			failed := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page := r.URL.Query().Get("pageToken")
				pages = append(pages, page)
				w.Header().Set("Content-Type", "application/json")
				if page == "second" && !failed {
					failed = true
					w.WriteHeader(403)
					_, _ = w.Write([]byte(`{"error":{"code":403,"errors":[{"reason":"userRateLimitExceeded"}]}}`))
					return
				}
				if page == "" {
					_, _ = w.Write([]byte(`{"nextPageToken":"second","items":[{"id":"a","status":"cancelled","title":"a"}]}`))
				} else {
					_, _ = w.Write([]byte(`{"nextSyncToken":"new","items":[{"id":"b","status":"cancelled","title":"b"}]}`))
				}
			}))
			defer server.Close()
			svc, err := calendar.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/"))
			require.NoError(t, err)
			ts, err := gtasks.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/"))
			require.NoError(t, err)
			q, _ := fakeQuota()
			p := &Provider{svc: svc, tasksSvc: ts, quota: q}
			c := cal.Calendar{RemoteID: "test"}
			if tasks {
				result, err := p.syncTasks(context.Background(), c)
				require.NoError(t, err)
				require.Len(t, result.TaskChanges, 2)
			} else {
				changes, cursor, err := p.syncPages(context.Background(), c, "old")
				require.NoError(t, err)
				require.Len(t, changes, 2)
				require.Equal(t, "new", cursor)
			}
			require.Equal(t, []string{"", "second", "second"}, pages)
		})
	}
}

func TestMutationIsNeverRetried(t *testing.T) {
	q, _ := fakeQuota()
	p := &Provider{quota: q}
	calls := 0
	_, err := googleCall(context.Background(), p, false, func() (*int, error) { calls++; return nil, &googleapi.Error{Code: 429} })
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

type fixtureSecrets struct{}

func (fixtureSecrets) Get(_ context.Context, _, key string) ([]byte, error) {
	if key == SecretKeyApp {
		return []byte(`{"client_id":"fixture","client_secret":"fixture"}`), nil
	}
	return []byte(`{"access_token":"fixture","token_type":"Bearer"}`), nil
}
func (fixtureSecrets) Set(context.Context, string, string, []byte) error { return nil }
func (fixtureSecrets) Delete(context.Context, string, string) error      { return nil }

func TestFactoryRebuildSharesBudgetAndLongCooldown(t *testing.T) {
	ctx := context.Background()
	id := t.Name()
	q, sleeps := fakeQuota()
	accountQuotas.Store(id, q)
	t.Cleanup(func() { accountQuotas.Delete(id) })
	build := func(id string) *Provider {
		provider, err := (Factory{}).Build(ctx, cal.Account{ID: id, Kind: cal.AccountGoogle}, fixtureSecrets{})
		require.NoError(t, err)
		return provider.(*Provider)
	}
	first := build(id)
	require.NoError(t, first.waitQuota(ctx))
	_ = first.Close()
	second := build(id)
	require.Same(t, first.quotaGate(), second.quotaGate(), "rebuilt provider lost account gate")
	require.NoError(t, second.waitQuota(ctx))
	require.Equal(t, []time.Duration{time.Second / 4}, *sleeps, "pacing not shared")
	calls := 0
	_, err := googleCall(ctx, second, true, func() (*int, error) {
		calls++
		return nil, &googleapi.Error{Code: 429, Header: http.Header{"Retry-After": []string{"3600"}}}
	})
	var deferred *deferredRetry
	require.ErrorAs(t, err, &deferred)
	require.GreaterOrEqual(t, deferred.RetryAfter(), time.Hour)
	require.Equal(t, 1, calls)
	before := len(*sleeps)
	third := build(id)
	require.ErrorAs(t, third.waitQuota(ctx), &deferred, "cooldown lost")
	require.Len(t, *sleeps, before, "rebuilt provider slept for long cooldown")
	otherID := id + "-other"
	t.Cleanup(func() { accountQuotas.Delete(otherID) })
	require.NoError(t, build(otherID).waitQuota(ctx), "other account blocked")
}

func TestRetryWaitBudgetSpansPages(t *testing.T) {
	q, sleeps := fakeQuota()
	p := &Provider{quota: q}
	deferred := false
	for page := 0; page < 10; page++ {
		calls := 0
		_, err := googleCall(context.Background(), p, true, func() (*int, error) {
			calls++
			if calls == 1 {
				return nil, &googleapi.Error{Code: 429}
			}
			n := 1
			return &n, nil
		})
		if err != nil {
			var d *deferredRetry
			require.ErrorAs(t, err, &d)
			deferred = true
			break
		}
	}
	require.True(t, deferred, "retry budget reset across pages")
	require.LessOrEqual(t, p.retryWait, maxInlineRetryWait)
	require.NotEmpty(t, *sleeps, "short retry was not exercised")
}
