package calendar_handler

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/AvengeMedia/dankcalendar/core/api/server"
	"github.com/AvengeMedia/dankcalendar/core/ent"
	"github.com/AvengeMedia/dankcalendar/core/errdefs"
	"github.com/AvengeMedia/dankcalendar/core/models"
	"github.com/AvengeMedia/dankcalendar/core/repo"
)

type HandlerGroup struct {
	srv *server.Server
}

func RegisterHandlers(srv *server.Server, grp *huma.Group) {
	h := &HandlerGroup{srv: srv}

	huma.Register(grp, huma.Operation{
		OperationID: "list-accounts",
		Summary:     "List Accounts",
		Path:        "/accounts",
		Method:      http.MethodGet,
	}, h.ListAccounts)

	huma.Register(grp, huma.Operation{
		OperationID: "list-calendars",
		Summary:     "List Calendars",
		Path:        "/calendars",
		Method:      http.MethodGet,
	}, h.ListCalendars)

	huma.Register(grp, huma.Operation{
		OperationID: "list-events",
		Summary:     "List Events",
		Path:        "/events",
		Method:      http.MethodGet,
	}, h.ListEvents)

	huma.Register(grp, huma.Operation{
		OperationID: "get-event",
		Summary:     "Get Event",
		Path:        "/events/{id}",
		Method:      http.MethodGet,
	}, h.GetEvent)
}

type ListAccountsOutput struct {
	Body struct {
		Accounts []models.Account `json:"accounts"`
	}
}

func (h *HandlerGroup) ListAccounts(ctx context.Context, _ *server.EmptyInput) (*ListAccountsOutput, error) {
	accounts, err := h.srv.Repo.ListAccounts(ctx)
	if err != nil {
		return nil, errdefs.NewCustomError(errdefs.ErrTypeProvider, err.Error())
	}

	out := &ListAccountsOutput{}
	out.Body.Accounts = make([]models.Account, 0, len(accounts))
	for _, a := range accounts {
		out.Body.Accounts = append(out.Body.Accounts, accountToDTO(a))
	}
	return out, nil
}

type ListCalendarsInput struct {
	AccountID string `query:"accountId"`
}

type ListCalendarsOutput struct {
	Body struct {
		Calendars []models.Calendar `json:"calendars"`
	}
}

func (h *HandlerGroup) ListCalendars(ctx context.Context, in *ListCalendarsInput) (*ListCalendarsOutput, error) {
	var (
		calendars []*ent.Calendar
		err       error
	)
	switch {
	case in.AccountID != "":
		calendars, err = h.srv.Repo.ListCalendarsForAccount(ctx, in.AccountID)
	default:
		calendars, err = h.srv.Repo.ListCalendars(ctx)
	}
	if err != nil {
		return nil, errdefs.NewCustomError(errdefs.ErrTypeProvider, err.Error())
	}

	out := &ListCalendarsOutput{}
	out.Body.Calendars = make([]models.Calendar, 0, len(calendars))
	for _, c := range calendars {
		out.Body.Calendars = append(out.Body.Calendars, calendarToDTO(c))
	}
	return out, nil
}

type ListEventsInput struct {
	CalendarIDs []string `query:"calendarId"`
	From        string   `query:"from"`
	To          string   `query:"to"`
	Query       string   `query:"q"`
	Limit       int      `query:"limit"`
	Offset      int      `query:"offset"`
}

type ListEventsOutput struct {
	Body models.EventList
}

func (h *HandlerGroup) ListEvents(ctx context.Context, in *ListEventsInput) (*ListEventsOutput, error) {
	filter := repo.EventFilter{
		CalendarIDs:      in.CalendarIDs,
		Query:            in.Query,
		IncludeRecurring: true,
	}

	if in.From != "" {
		t, err := time.Parse(time.RFC3339, in.From)
		if err != nil {
			return nil, errdefs.NewCustomError(errdefs.ErrTypeInvalidInput, "invalid from timestamp")
		}
		filter.From = &t
	}
	if in.To != "" {
		t, err := time.Parse(time.RFC3339, in.To)
		if err != nil {
			return nil, errdefs.NewCustomError(errdefs.ErrTypeInvalidInput, "invalid to timestamp")
		}
		filter.To = &t
	}

	events, total, err := h.srv.Repo.ListEvents(ctx, repo.ListEventsParams{
		Filter: filter,
		Limit:  in.Limit,
		Offset: in.Offset,
	})
	if err != nil {
		return nil, errdefs.NewCustomError(errdefs.ErrTypeProvider, err.Error())
	}

	out := &ListEventsOutput{}
	out.Body.Total = total
	out.Body.Events = make([]models.Event, 0, len(events))
	for _, e := range events {
		out.Body.Events = append(out.Body.Events, eventToDTO(e))
	}
	return out, nil
}

type GetEventInput struct {
	ID string `path:"id"`
}

type GetEventOutput struct {
	Body models.Event
}

func (h *HandlerGroup) GetEvent(ctx context.Context, in *GetEventInput) (*GetEventOutput, error) {
	event, err := h.srv.Repo.GetEvent(ctx, in.ID)
	if err != nil {
		if repo.IsNotFound(err) {
			return nil, errdefs.NewCustomError(errdefs.ErrTypeNotFound, "event not found")
		}
		return nil, errdefs.NewCustomError(errdefs.ErrTypeProvider, err.Error())
	}
	return &GetEventOutput{Body: eventToDTO(event)}, nil
}

func accountToDTO(a *ent.Account) models.Account {
	return models.Account{
		ID:          a.ID,
		Kind:        string(a.Kind),
		DisplayName: a.DisplayName,
		Settings:    a.Settings,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

func calendarToDTO(c *ent.Calendar) models.Calendar {
	accountID := ""
	if edges, err := c.Edges.AccountOrErr(); err == nil && edges != nil {
		accountID = edges.ID
	}
	return models.Calendar{
		ID:          c.ID,
		AccountID:   accountID,
		RemoteID:    c.RemoteID,
		Name:        c.Name,
		Description: c.Description,
		Color:       c.Color,
		TimeZone:    c.TimeZone,
		ReadOnly:    c.ReadOnly,
		Hidden:      c.Hidden,
		UpdatedAt:   c.UpdatedAt,
	}
}

func eventToDTO(e *ent.Event) models.Event {
	calendarID := ""
	if edges, err := e.Edges.CalendarOrErr(); err == nil && edges != nil {
		calendarID = edges.ID
	}
	return models.Event{
		ID:            e.ID,
		CalendarID:    calendarID,
		UID:           e.UID,
		Summary:       e.Summary,
		Description:   e.Description,
		Location:      e.Location,
		URL:           e.URL,
		Status:        string(e.Status),
		Start:         e.Start,
		End:           e.End,
		AllDay:        e.AllDay,
		StartTimeZone: e.StartTz,
		EndTimeZone:   e.EndTz,
		RecurringID:   e.RecurringID,
	}
}
