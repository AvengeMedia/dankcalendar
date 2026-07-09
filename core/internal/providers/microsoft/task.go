package microsoft

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	cal "github.com/AvengeMedia/dankcalendar/core/internal/calendar"
)

// todoListPrefix namespaces To Do list remote ids so they cannot collide with
// event-calendar ids under the same account's unique (account, remote_id) index.
const todoListPrefix = "todolist:"

type graphTodoList struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type graphTodoTask struct {
	ID                string         `json:"id,omitempty"`
	Title             string         `json:"title"`
	Body              *graphBody     `json:"body,omitempty"`
	Status            string         `json:"status,omitempty"`
	Importance        string         `json:"importance,omitempty"`
	CompletedDateTime *graphDateTime `json:"completedDateTime,omitempty"`
	// Sent unconditionally (no omitempty) so a PATCH can clear them; omitting a
	// field leaves Graph's stored value intact.
	DueDateTime *graphDateTime            `json:"dueDateTime"`
	Recurrence  *graphPatternedRecurrence `json:"recurrence"`
}

type graphPatternedRecurrence struct {
	Pattern graphRecurrencePattern `json:"pattern"`
	Range   graphRecurrenceRange   `json:"range"`
}

type graphRecurrencePattern struct {
	Type       string   `json:"type"`
	Interval   int      `json:"interval"`
	DaysOfWeek []string `json:"daysOfWeek,omitempty"`
	DayOfMonth int      `json:"dayOfMonth,omitempty"`
	Month      int      `json:"month,omitempty"`
}

type graphRecurrenceRange struct {
	Type                string `json:"type"`
	StartDate           string `json:"startDate,omitempty"`
	EndDate             string `json:"endDate,omitempty"`
	NumberOfOccurrences int    `json:"numberOfOccurrences,omitempty"`
	RecurrenceTimeZone  string `json:"recurrenceTimeZone,omitempty"`
}

func (p *Provider) listTodoLists(ctx context.Context) ([]cal.Calendar, error) {
	var out []cal.Calendar
	next := graphBase + "/me/todo/lists?$top=100"
	for next != "" {
		var page struct {
			Value    []graphTodoList `json:"value"`
			NextLink string          `json:"@odata.nextLink"`
		}
		if err := p.doJSON(ctx, http.MethodGet, next, nil, &page); err != nil {
			return nil, fmt.Errorf("list microsoft task lists: %w", classifyAuthErr(err))
		}
		for _, item := range page.Value {
			out = append(out, cal.Calendar{
				AccountID:           p.account.ID,
				RemoteID:            todoListPrefix + item.ID,
				Name:                item.DisplayName,
				SupportedComponents: []string{cal.ComponentVTodo},
			})
		}
		next = page.NextLink
	}
	return out, nil
}

func (p *Provider) syncTasks(ctx context.Context, c cal.Calendar) (*cal.SyncResult, error) {
	listID := todoListID(c.RemoteID)
	var changes []cal.TaskChange
	next := graphBase + "/me/todo/lists/" + url.PathEscape(listID) + "/tasks?$top=100"
	for next != "" {
		var page struct {
			Value    []graphTodoTask `json:"value"`
			NextLink string          `json:"@odata.nextLink"`
		}
		if err := p.doJSON(ctx, http.MethodGet, next, nil, &page); err != nil {
			return nil, fmt.Errorf("sync microsoft tasks: %w", classifyAuthErr(err))
		}
		for _, item := range page.Value {
			task := graphToTask(c, item)
			changes = append(changes, cal.TaskChange{Type: cal.ChangeUpsert, Task: &task})
		}
		next = page.NextLink
	}

	return &cal.SyncResult{
		Cursor:       cal.SyncCursor{CalendarID: c.ID},
		TaskChanges:  changes,
		FullSnapshot: true,
	}, nil
}

func (p *Provider) CreateTask(ctx context.Context, c cal.Calendar, t *cal.Task) (*cal.Task, error) {
	var created graphTodoTask
	reqURL := graphBase + "/me/todo/lists/" + url.PathEscape(todoListID(c.RemoteID)) + "/tasks"
	if err := p.doJSON(ctx, http.MethodPost, reqURL, taskToGraph(t), &created); err != nil {
		return nil, fmt.Errorf("microsoft create task: %w", err)
	}
	out := graphToTask(c, created)
	return &out, nil
}

func (p *Provider) UpdateTask(ctx context.Context, c cal.Calendar, t *cal.Task) (*cal.Task, error) {
	if t.RemoteID == "" {
		return nil, errors.New("microsoft update task: missing remote id")
	}
	var updated graphTodoTask
	reqURL := graphBase + "/me/todo/lists/" + url.PathEscape(todoListID(c.RemoteID)) + "/tasks/" + url.PathEscape(t.RemoteID)
	if err := p.doJSON(ctx, http.MethodPatch, reqURL, taskToGraph(t), &updated); err != nil {
		return nil, fmt.Errorf("microsoft update task: %w", err)
	}
	out := graphToTask(c, updated)
	return &out, nil
}

func (p *Provider) DeleteTask(ctx context.Context, c cal.Calendar, t cal.Task) error {
	if t.RemoteID == "" {
		return errors.New("microsoft delete task: missing remote id")
	}
	reqURL := graphBase + "/me/todo/lists/" + url.PathEscape(todoListID(c.RemoteID)) + "/tasks/" + url.PathEscape(t.RemoteID)
	err := p.doJSON(ctx, http.MethodDelete, reqURL, nil, nil)
	var ge *graphError
	if errors.As(err, &ge) && ge.Status == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("microsoft delete task: %w", err)
	}
	return nil
}

func todoListID(remoteID string) string {
	return strings.TrimPrefix(remoteID, todoListPrefix)
}

// graphToTask maps a To Do task. Graph carries no percent-complete and its due
// value has date precision only, so a due task is treated as all-day.
func graphToTask(c cal.Calendar, g graphTodoTask) cal.Task {
	t := cal.Task{
		CalendarID: c.ID,
		UID:        g.ID,
		RemoteID:   g.ID,
		Summary:    g.Title,
		Status:     fromGraphTaskStatus(g.Status),
		Priority:   fromGraphImportance(g.Importance),
	}
	t.Description = graphBodyText(g.Body)
	if t.Status == cal.TaskCompleted {
		t.PercentComplete = 100
	}
	if g.DueDateTime != nil && g.DueDateTime.DateTime != "" {
		t.Due = parseGraphTime(g.DueDateTime.DateTime)
		t.AllDay = true
	}
	if g.CompletedDateTime != nil && g.CompletedDateTime.DateTime != "" {
		t.Completed = parseGraphTime(g.CompletedDateTime.DateTime)
	}
	if rrule := rruleFromGraphRecurrence(g.Recurrence); rrule != "" {
		t.Recurrence = &cal.Recurrence{RRule: []string{rrule}}
	}
	return t
}

func taskToGraph(t *cal.Task) graphTodoTask {
	g := graphTodoTask{
		Title:      t.Summary,
		Body:       &graphBody{ContentType: "text", Content: t.Description},
		Status:     toGraphTaskStatus(t.Status),
		Importance: toGraphImportance(t.Priority),
	}
	if !t.Due.IsZero() {
		g.DueDateTime = &graphDateTime{DateTime: t.Due.UTC().Format(graphTimeLayout), TimeZone: "UTC"}
	}
	g.Recurrence = graphRecurrenceFromTask(t)
	return g
}

var graphWeekdays = []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

var rruleWeekdays = map[time.Weekday]string{
	time.Sunday: "SU", time.Monday: "MO", time.Tuesday: "TU", time.Wednesday: "WE",
	time.Thursday: "TH", time.Friday: "FR", time.Saturday: "SA",
}

// rruleFromGraphRecurrence renders a Graph patterned recurrence as an iCalendar
// RRULE so it stores and displays identically to CalDAV tasks. Only FREQ,
// INTERVAL, BYDAY (weekly) and COUNT are carried — the subset the editor offers.
func rruleFromGraphRecurrence(r *graphPatternedRecurrence) string {
	if r == nil {
		return ""
	}
	var freq string
	switch r.Pattern.Type {
	case "daily":
		freq = "DAILY"
	case "weekly":
		freq = "WEEKLY"
	case "absoluteMonthly", "relativeMonthly":
		freq = "MONTHLY"
	case "absoluteYearly", "relativeYearly":
		freq = "YEARLY"
	default:
		return ""
	}

	parts := []string{"FREQ=" + freq}
	if r.Pattern.Interval > 1 {
		parts = append(parts, fmt.Sprintf("INTERVAL=%d", r.Pattern.Interval))
	}
	if freq == "WEEKLY" && len(r.Pattern.DaysOfWeek) > 0 {
		var days []string
		for _, d := range r.Pattern.DaysOfWeek {
			if code := rruleWeekdayFromGraph(d); code != "" {
				days = append(days, code)
			}
		}
		if len(days) > 0 {
			parts = append(parts, "BYDAY="+strings.Join(days, ","))
		}
	}
	if r.Range.Type == "numbered" && r.Range.NumberOfOccurrences > 0 {
		parts = append(parts, fmt.Sprintf("COUNT=%d", r.Range.NumberOfOccurrences))
	}
	return strings.Join(parts, ";")
}

func rruleWeekdayFromGraph(day string) string {
	for i, name := range graphWeekdays {
		if name == day {
			return rruleWeekdays[time.Weekday(i)]
		}
	}
	return ""
}

// graphRecurrenceFromTask converts the task's RRULE into a Graph patterned
// recurrence. Graph anchors recurrence on a start date and needs concrete
// day/month values, so absent any due date (the anchor) recurrence is dropped.
func graphRecurrenceFromTask(t *cal.Task) *graphPatternedRecurrence {
	if t.Recurrence == nil || len(t.Recurrence.RRule) == 0 || t.Due.IsZero() {
		return nil
	}
	rule := parseRRule(t.Recurrence.RRule[0])
	due := t.Due.UTC()

	pattern := graphRecurrencePattern{Interval: 1}
	if rule["INTERVAL"] != "" {
		if n, err := strconv.Atoi(rule["INTERVAL"]); err == nil && n > 0 {
			pattern.Interval = n
		}
	}
	switch rule["FREQ"] {
	case "DAILY":
		pattern.Type = "daily"
	case "WEEKLY":
		pattern.Type = "weekly"
		pattern.DaysOfWeek = graphWeekdaysFromRRule(rule["BYDAY"], due.Weekday())
	case "MONTHLY":
		pattern.Type = "absoluteMonthly"
		pattern.DayOfMonth = due.Day()
	case "YEARLY":
		pattern.Type = "absoluteYearly"
		pattern.Month = int(due.Month())
		pattern.DayOfMonth = due.Day()
	default:
		return nil
	}

	rng := graphRecurrenceRange{Type: "noEnd", StartDate: due.Format("2006-01-02")}
	if rule["COUNT"] != "" {
		if n, err := strconv.Atoi(rule["COUNT"]); err == nil && n > 0 {
			rng.Type = "numbered"
			rng.NumberOfOccurrences = n
		}
	}
	return &graphPatternedRecurrence{Pattern: pattern, Range: rng}
}

func graphWeekdaysFromRRule(byday string, fallback time.Weekday) []string {
	codeToDay := map[string]string{
		"SU": "sunday", "MO": "monday", "TU": "tuesday", "WE": "wednesday",
		"TH": "thursday", "FR": "friday", "SA": "saturday",
	}
	var out []string
	for _, code := range strings.Split(byday, ",") {
		if day, ok := codeToDay[strings.TrimSpace(code)]; ok {
			out = append(out, day)
		}
	}
	if len(out) == 0 {
		out = append(out, graphWeekdays[fallback])
	}
	return out
}

// parseRRule splits an "FREQ=…;INTERVAL=…" rule into its key/value parts.
func parseRRule(rule string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(rule, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			out[strings.ToUpper(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
		}
	}
	return out
}

func fromGraphTaskStatus(s string) cal.TaskStatus {
	switch s {
	case "inProgress", "waitingOnOthers":
		return cal.TaskInProcess
	case "completed":
		return cal.TaskCompleted
	default:
		return cal.TaskNeedsAction
	}
}

func toGraphTaskStatus(s cal.TaskStatus) string {
	switch s {
	case cal.TaskInProcess:
		return "inProgress"
	case cal.TaskCompleted:
		return "completed"
	default:
		return "notStarted"
	}
}

// fromGraphImportance maps Graph's three-level importance onto the VTODO 1-9
// priority scale (1 highest), leaving normal unscored so it round-trips cleanly.
func fromGraphImportance(s string) int {
	switch s {
	case "high":
		return 1
	case "low":
		return 9
	default:
		return 0
	}
}

func toGraphImportance(priority int) string {
	switch {
	case priority == 0:
		return "normal"
	case priority <= 4:
		return "high"
	case priority >= 6:
		return "low"
	default:
		return "normal"
	}
}
