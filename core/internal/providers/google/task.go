package google

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
	gtasks "google.golang.org/api/tasks/v1"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

// taskListPrefix namespaces task-list remote ids so they cannot collide with
// event-calendar ids under the same account's unique (account, remote_id) index.
const taskListPrefix = "tasklist:"

func (p *Provider) listTaskLists(ctx context.Context) ([]cal.Calendar, error) {
	res, err := p.tasksSvc.Tasklists.List().Context(ctx).MaxResults(100).Do()
	if err != nil {
		return nil, fmt.Errorf("list google task lists: %w", classifyAuthErr(err))
	}

	out := make([]cal.Calendar, 0, len(res.Items))
	for _, item := range res.Items {
		out = append(out, cal.Calendar{
			AccountID:           p.account.ID,
			RemoteID:            taskListPrefix + item.Id,
			Name:                item.Title,
			SupportedComponents: []string{cal.ComponentVTodo},
		})
	}
	return out, nil
}

func (p *Provider) syncTasks(ctx context.Context, c cal.Calendar) (*cal.SyncResult, error) {
	listID := taskListID(c.RemoteID)
	var (
		changes   []cal.TaskChange
		pageToken string
	)
	for {
		call := p.tasksSvc.Tasks.List(listID).
			Context(ctx).
			ShowCompleted(true).
			ShowHidden(true).
			MaxResults(100)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		res, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("sync google tasks: %w", classifyAuthErr(err))
		}
		for _, item := range res.Items {
			changes = append(changes, cal.TaskChange{Type: cal.ChangeUpsert, Task: fromGoogleTask(c, item)})
		}
		if res.NextPageToken == "" {
			break
		}
		pageToken = res.NextPageToken
	}

	return &cal.SyncResult{
		Cursor:       cal.SyncCursor{CalendarID: c.ID},
		TaskChanges:  changes,
		FullSnapshot: true,
	}, nil
}

func (p *Provider) CreateTask(ctx context.Context, c cal.Calendar, t *cal.Task) (*cal.Task, error) {
	created, err := p.tasksSvc.Tasks.Insert(taskListID(c.RemoteID), toGoogleTask(t)).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("create google task: %w", err)
	}
	return fromGoogleTask(c, created), nil
}

func (p *Provider) UpdateTask(ctx context.Context, c cal.Calendar, t *cal.Task) (*cal.Task, error) {
	if t.RemoteID == "" {
		return nil, errors.New("update google task: missing remote id")
	}
	// tasks.update validates the id in the request body, not just the path, so
	// it must be set or the API rejects the call with "Missing task ID".
	body := toGoogleTask(t)
	body.Id = t.RemoteID
	updated, err := p.tasksSvc.Tasks.Update(taskListID(c.RemoteID), t.RemoteID, body).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("update google task: %w", err)
	}
	return fromGoogleTask(c, updated), nil
}

func (p *Provider) DeleteTask(ctx context.Context, c cal.Calendar, t cal.Task) error {
	if t.RemoteID == "" {
		return errors.New("delete google task: missing remote id")
	}
	err := p.tasksSvc.Tasks.Delete(taskListID(c.RemoteID), t.RemoteID).Context(ctx).Do()
	if err == nil {
		return nil
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && (apiErr.Code == http.StatusNotFound || apiErr.Code == http.StatusGone) {
		return nil
	}
	return fmt.Errorf("delete google task: %w", err)
}

func taskListID(remoteID string) string {
	return strings.TrimPrefix(remoteID, taskListPrefix)
}

// fromGoogleTask maps a Google task. The API exposes no priority or
// percent-complete, and its due value carries date precision only, so a due task
// is treated as all-day.
func fromGoogleTask(c cal.Calendar, item *gtasks.Task) *cal.Task {
	t := &cal.Task{
		CalendarID:  c.ID,
		UID:         item.Id,
		RemoteID:    item.Id,
		Etag:        item.Etag,
		Summary:     item.Title,
		Description: item.Notes,
		Status:      fromGoogleTaskStatus(item.Status),
		ParentUID:   item.Parent,
	}
	if t.Status == cal.TaskCompleted {
		t.PercentComplete = 100
	}
	if item.Due != "" {
		if due, err := time.Parse(time.RFC3339, item.Due); err == nil {
			t.Due = due
			t.AllDay = true
		}
	}
	if item.Completed != nil && *item.Completed != "" {
		t.Completed, _ = time.Parse(time.RFC3339, *item.Completed)
	}
	return t
}

func toGoogleTask(t *cal.Task) *gtasks.Task {
	out := &gtasks.Task{
		Title:  t.Summary,
		Notes:  t.Description,
		Status: toGoogleTaskStatus(t.Status),
	}
	if !t.Due.IsZero() {
		out.Due = t.Due.UTC().Format(time.RFC3339)
	}
	// Clearing the due date or completion must blank the field server-side.
	if t.Due.IsZero() {
		out.NullFields = append(out.NullFields, "Due")
	}
	if t.Status != cal.TaskCompleted {
		out.NullFields = append(out.NullFields, "Completed")
	}
	return out
}

func fromGoogleTaskStatus(s string) cal.TaskStatus {
	if s == "completed" {
		return cal.TaskCompleted
	}
	return cal.TaskNeedsAction
}

func toGoogleTaskStatus(s cal.TaskStatus) string {
	if s == cal.TaskCompleted {
		return "completed"
	}
	return "needsAction"
}
