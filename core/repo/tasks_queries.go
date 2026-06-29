package repo

import (
	"context"

	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/ent/calendar"
	"github.com/AvengeMedia/dankcalendar/core/ent/predicate"
	"github.com/AvengeMedia/dankcalendar/core/ent/task"
	"github.com/AvengeMedia/dankcalendar/core/errdefs"
)

type TaskFilter struct {
	CalendarIDs      []string
	IncludeCompleted bool
	Query            string
}

type ListTasksParams struct {
	Filter TaskFilter
	Limit  int
	Offset int
}

func (r *Repo) GetTask(ctx context.Context, id string) (*ent.Task, error) {
	t, err := r.client.Task.Query().Where(task.IDEQ(id)).WithCalendar().Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return nil, errdefs.NewCustomError(errdefs.ErrTypeNotFound, "task not found")
	case err != nil:
		return nil, err
	}
	return t, nil
}

func (r *Repo) ListTasks(ctx context.Context, p ListTasksParams) ([]*ent.Task, int, error) {
	q := r.client.Task.Query().Where(taskPredicates(p.Filter)...)

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	q = q.WithCalendar().Order(ent.Asc(task.FieldDue), ent.Asc(task.FieldSummary))
	if p.Limit > 0 {
		q = q.Limit(p.Limit)
	}
	if p.Offset > 0 {
		q = q.Offset(p.Offset)
	}

	tasks, err := q.All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (r *Repo) FindTaskByUID(ctx context.Context, calendarID, uid string) (*ent.Task, error) {
	t, err := r.client.Task.Query().
		Where(
			task.HasCalendarWith(calendar.IDEQ(calendarID)),
			task.UIDEQ(uid),
		).
		Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return nil, errdefs.NewCustomError(errdefs.ErrTypeNotFound, "task not found")
	case err != nil:
		return nil, err
	}
	return t, nil
}

func taskPredicates(f TaskFilter) []predicate.Task {
	var preds []predicate.Task
	if len(f.CalendarIDs) > 0 {
		preds = append(preds, task.HasCalendarWith(calendar.IDIn(f.CalendarIDs...)))
	}
	if !f.IncludeCompleted {
		preds = append(preds, task.StatusNEQ(task.StatusCompleted), task.StatusNEQ(task.StatusCancelled))
	}
	if f.Query != "" {
		preds = append(preds, task.Or(
			task.SummaryContainsFold(f.Query),
			task.DescriptionContainsFold(f.Query),
			task.LocationContainsFold(f.Query),
		))
	}
	return preds
}
