package google

import (
	"context"
	"errors"
	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	gtasks "google.golang.org/api/tasks/v1"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"google.golang.org/api/googleapi"
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
		if err := q.wait(ctx, cost); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(*sleeps, []time.Duration{time.Second, time.Second / 2}) {
		t.Fatalf("waits %v", *sleeps)
	}
	q.cooldown(time.Minute)
	q.cooldown(time.Second)
	if err := q.wait(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if got := (*sleeps)[2]; got != time.Minute {
		t.Fatalf("cooldown shortened: %v", got)
	}
}

func TestReadRetryDelay(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	cases := []struct {
		name  string
		err   error
		want  time.Duration
		retry bool
	}{
		{"quota", &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "rateLimitExceeded"}}}, time.Minute, true},
		{"user quota", &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "userRateLimitExceeded"}}}, time.Minute, true},
		{"429", &googleapi.Error{Code: 429}, time.Minute, true},
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
			if d != tt.want || retry != tt.retry {
				t.Fatalf("got %v,%v", d, retry)
			}
		})
	}
	if d, _ := readRetryDelay(&googleapi.Error{Code: 429}, 4, now); d != 2*time.Minute {
		t.Fatalf("backoff %v", d)
	}
}

func TestReadRetriesAreBoundedAndCancellable(t *testing.T) {
	q, _ := fakeQuota()
	r := &Provider{quota: q}
	calls := 0
	_, err := googleCall(context.Background(), r, true, func() (*int, error) { calls++; return nil, &googleapi.Error{Code: 429} })
	if err == nil || calls != maxReadRetries+1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls = 0
	q, _ = fakeQuota()
	r = &Provider{quota: q}
	_, err = googleCall(ctx, r, true, func() (*int, error) { calls++; cancel(); return nil, &googleapi.Error{Code: 429} })
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
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
			if err != nil {
				t.Fatal(err)
			}
			ts, err := gtasks.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/"))
			if err != nil {
				t.Fatal(err)
			}
			q, _ := fakeQuota()
			p := &Provider{svc: svc, tasksSvc: ts, quota: q}
			c := cal.Calendar{RemoteID: "test"}
			if tasks {
				result, err := p.syncTasks(context.Background(), c)
				if err != nil {
					t.Fatal(err)
				}
				if len(result.TaskChanges) != 2 {
					t.Fatalf("changes %d", len(result.TaskChanges))
				}
			} else {
				changes, cursor, err := p.syncPages(context.Background(), c, "old")
				if err != nil {
					t.Fatal(err)
				}
				if len(changes) != 2 || cursor != "new" {
					t.Fatalf("changes=%d cursor=%s", len(changes), cursor)
				}
			}
			if !reflect.DeepEqual(pages, []string{"", "second", "second"}) {
				t.Fatalf("pages %v", pages)
			}
		})
	}
}

func TestMutationIsNeverRetried(t *testing.T) {
	q, _ := fakeQuota()
	p := &Provider{quota: q}
	calls := 0
	_, err := googleCall(context.Background(), p, false, func() (*int, error) { calls++; return nil, &googleapi.Error{Code: 429} })
	if err == nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
