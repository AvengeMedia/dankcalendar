package repo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AvengeMedia/dankcalendar/core/ent/account"
	"github.com/AvengeMedia/dankcalendar/core/ent/task"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

func TestUpsertTaskAndList(t *testing.T) {
	r, ctx := newTestRepo(t)

	_, err := r.CreateAccount(ctx, repo.CreateAccountInput{
		ID:          "personal",
		Kind:        account.KindCaldav,
		DisplayName: "Personal",
	})
	require.NoError(t, err)

	cal, err := r.UpsertCalendar(ctx, repo.UpsertCalendarInput{
		AccountID:           "personal",
		RemoteID:            "/tasks/",
		Name:                "Reminders",
		SupportedComponents: []string{"VTODO"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"VTODO"}, cal.SupportedComponents)

	due := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	created, err := r.UpsertTask(ctx, repo.UpsertTaskInput{
		CalendarID: cal.ID,
		UID:        "todo-1",
		Summary:    "Pay rent",
		Status:     task.StatusNeedsAction,
		Priority:   5,
		Due:        due,
	})
	require.NoError(t, err)
	require.Equal(t, "Pay rent", created.Summary)
	require.NotNil(t, created.Due)
	require.Equal(t, due, created.Due.UTC())

	// Open tasks are returned by default; completed ones are hidden.
	open, total, err := r.ListTasks(ctx, repo.ListTasksParams{
		Filter: repo.TaskFilter{CalendarIDs: []string{cal.ID}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, open, 1)

	// Completing the task is an upsert on the same (calendar, uid).
	done := time.Now().UTC().Truncate(time.Second)
	updated, err := r.UpsertTask(ctx, repo.UpsertTaskInput{
		CalendarID:      cal.ID,
		UID:             "todo-1",
		Summary:         "Pay rent",
		Status:          task.StatusCompleted,
		PercentComplete: 100,
		Completed:       done,
	})
	require.NoError(t, err)
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, task.StatusCompleted, updated.Status)
	require.NotNil(t, updated.Completed)
	// Clearing the due date on update must null the column.
	require.Nil(t, updated.Due)

	_, openTotal, err := r.ListTasks(ctx, repo.ListTasksParams{
		Filter: repo.TaskFilter{CalendarIDs: []string{cal.ID}},
	})
	require.NoError(t, err)
	require.Zero(t, openTotal, "completed task should be hidden by default")

	_, allTotal, err := r.ListTasks(ctx, repo.ListTasksParams{
		Filter: repo.TaskFilter{CalendarIDs: []string{cal.ID}, IncludeCompleted: true},
	})
	require.NoError(t, err)
	require.Equal(t, 1, allTotal)
}

func TestDeleteTasksNotInUIDs(t *testing.T) {
	r, ctx := newTestRepo(t)

	_, err := r.CreateAccount(ctx, repo.CreateAccountInput{
		ID:          "acc",
		Kind:        account.KindCaldav,
		DisplayName: "Acc",
	})
	require.NoError(t, err)

	cal, err := r.UpsertCalendar(ctx, repo.UpsertCalendarInput{
		AccountID:           "acc",
		RemoteID:            "/tasks/",
		Name:                "Tasks",
		SupportedComponents: []string{"VTODO"},
	})
	require.NoError(t, err)

	for _, uid := range []string{"a", "b", "c"} {
		_, err := r.UpsertTask(ctx, repo.UpsertTaskInput{
			CalendarID: cal.ID,
			UID:        uid,
			Summary:    uid,
		})
		require.NoError(t, err)
	}

	// A full snapshot that only lists "a" prunes the rest.
	pruned, err := r.DeleteTasksNotInUIDs(ctx, cal.ID, []string{"a"})
	require.NoError(t, err)
	require.Equal(t, 2, pruned)

	_, total, err := r.ListTasks(ctx, repo.ListTasksParams{
		Filter: repo.TaskFilter{CalendarIDs: []string{cal.ID}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, total)
}

func TestDeleteCalendarCascadesTasks(t *testing.T) {
	r, ctx := newTestRepo(t)

	_, err := r.CreateAccount(ctx, repo.CreateAccountInput{
		ID:          "acc",
		Kind:        account.KindCaldav,
		DisplayName: "Acc",
	})
	require.NoError(t, err)

	cal, err := r.UpsertCalendar(ctx, repo.UpsertCalendarInput{
		AccountID:           "acc",
		RemoteID:            "/tasks/",
		Name:                "Tasks",
		SupportedComponents: []string{"VTODO"},
	})
	require.NoError(t, err)

	_, err = r.UpsertTask(ctx, repo.UpsertTaskInput{
		CalendarID: cal.ID,
		UID:        "todo-1",
		Summary:    "Task",
	})
	require.NoError(t, err)

	require.NoError(t, r.DeleteCalendar(ctx, cal.ID))

	_, total, err := r.ListTasks(ctx, repo.ListTasksParams{
		Filter: repo.TaskFilter{IncludeCompleted: true},
	})
	require.NoError(t, err)
	require.Zero(t, total)
}
